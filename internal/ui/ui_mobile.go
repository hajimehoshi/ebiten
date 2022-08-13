// Copyright 2016 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (android || ios) && !nintendosdk
// +build android ios
// +build !nintendosdk

package ui

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unicode"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

var (
	glContextCh = make(chan gl.Context, 1)

	// renderCh receives when updating starts.
	renderCh = make(chan struct{})

	// renderEndCh receives when updating finishes.
	renderEndCh = make(chan struct{})
)

func init() {
	theUI.userInterfaceImpl = userInterfaceImpl{
		foreground: 1,
		errCh:      make(chan error),

		// Give a default outside size so that the game can start without initializing them.
		outsideWidth:  640,
		outsideHeight: 480,
	}
	theUI.input.ui = &theUI.userInterfaceImpl
}

// Update is called from mobile/ebitenmobileview.
//
// Update must be called on the rendering thread.
func (u *userInterfaceImpl) Update() error {
	select {
	case err := <-u.errCh:
		return err
	default:
	}

	if !u.IsFocused() {
		return nil
	}

	if err := gamepad.Update(); err != nil {
		return err
	}

	renderCh <- struct{}{}
	go func() {
		<-renderEndCh
		u.t.Stop()
	}()
	u.t.Loop()
	return nil
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	outsideWidth  float64
	outsideHeight float64

	foreground int32
	errCh      chan error

	// Used for gomobile-build
	gbuildWidthPx   int
	gbuildHeightPx  int
	setGBuildSizeCh chan struct{}
	once            sync.Once

	context *context

	input Input

	fpsMode         FPSModeType
	renderRequester RenderRequester

	t *thread.OSThread

	m sync.RWMutex
}

func deviceScale() float64 {
	return devicescale.GetAt(0, 0)
}

// appMain is the main routine for gomobile-build mode.
func (u *userInterfaceImpl) appMain(a app.App) {
	var glctx gl.Context
	var sizeInited bool

	touches := map[touch.Sequence]Touch{}
	keys := map[Key]struct{}{}

	for e := range a.Events() {
		var updateInput bool
		var runes []rune

		switch e := a.Filter(e).(type) {
		case lifecycle.Event:
			switch e.Crosses(lifecycle.StageVisible) {
			case lifecycle.CrossOn:
				if err := u.SetForeground(true); err != nil {
					// There are no other ways than panicking here.
					panic(err)
				}
				restorable.OnContextLost()
				glctx, _ = e.DrawContext.(gl.Context)
				// Assume that glctx is always a same instance.
				// Then, only once initializing should be enough.
				if glContextCh != nil {
					glContextCh <- glctx
					glContextCh = nil
				}
				a.Send(paint.Event{})
			case lifecycle.CrossOff:
				if err := u.SetForeground(false); err != nil {
					// There are no other ways than panicking here.
					panic(err)
				}
				glctx = nil
			}
		case size.Event:
			u.setGBuildSize(e.WidthPx, e.HeightPx)
			sizeInited = true
		case paint.Event:
			if !sizeInited {
				a.Send(paint.Event{})
				continue
			}
			if glctx == nil || e.External {
				continue
			}
			renderCh <- struct{}{}
			<-renderEndCh
			a.Publish()
			a.Send(paint.Event{})
		case touch.Event:
			if !sizeInited {
				continue
			}
			switch e.Type {
			case touch.TypeBegin, touch.TypeMove:
				s := deviceScale()
				x, y := float64(e.X)/s, float64(e.Y)/s
				// TODO: Is it ok to cast from int64 to int here?
				touches[e.Sequence] = Touch{
					ID: TouchID(e.Sequence),
					X:  int(x),
					Y:  int(y),
				}
			case touch.TypeEnd:
				delete(touches, e.Sequence)
			}
			updateInput = true
		case key.Event:
			k, ok := gbuildKeyToUIKey[e.Code]
			if ok {
				switch e.Direction {
				case key.DirPress, key.DirNone:
					keys[k] = struct{}{}
				case key.DirRelease:
					delete(keys, k)
				}
			}

			switch e.Direction {
			case key.DirPress, key.DirNone:
				if e.Rune != -1 && unicode.IsPrint(e.Rune) {
					runes = []rune{e.Rune}
				}
			}
			updateInput = true
		}

		if updateInput {
			var ts []Touch
			for _, t := range touches {
				ts = append(ts, t)
			}
			u.input.update(keys, runes, ts)
		}
	}
}

func (u *userInterfaceImpl) SetForeground(foreground bool) error {
	var v int32
	if foreground {
		v = 1
	}
	atomic.StoreInt32(&u.foreground, v)

	if foreground {
		return hooks.ResumeAudio()
	} else {
		return hooks.SuspendAudio()
	}
}

func (u *userInterfaceImpl) Run(game Game) error {
	u.setGBuildSizeCh = make(chan struct{})
	go func() {
		if err := u.run(game, true); err != nil {
			// As mobile apps never ends, Loop can't return. Just panic here.
			panic(err)
		}
	}()
	app.Main(u.appMain)
	return nil
}

func RunWithoutMainLoop(game Game) {
	theUI.runWithoutMainLoop(game)
}

func (u *userInterfaceImpl) runWithoutMainLoop(game Game) {
	go func() {
		if err := u.run(game, false); err != nil {
			u.errCh <- err
		}
	}()
}

func (u *userInterfaceImpl) run(game Game, mainloop bool) (err error) {
	// Convert the panic to a regular error so that Java/Objective-C layer can treat this easily e.g., for
	// Crashlytics. A panic is treated as SIGABRT, and there is no way to handle this on Java/Objective-C layer
	// unfortunately.
	// TODO: Panic on other goroutines cannot be handled here.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v\n%s", r, string(debug.Stack()))
		}
	}()

	u.context = newContext(game)
	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		gomobileBuild: mainloop,
	})
	if err != nil {
		return err
	}
	u.graphicsDriver = g

	if mainloop {
		// When gomobile-build is used, GL functions must be called via
		// gl.Context so that they are called on the appropriate thread.
		ctx := <-glContextCh
		g.(*opengl.Graphics).SetGomobileGLContext(ctx)
	} else {
		u.t = thread.NewOSThread()
		graphicscommand.SetRenderingThread(u.t)
	}

	// If gomobile-build is used, wait for the outside size fixed.
	if u.setGBuildSizeCh != nil {
		<-u.setGBuildSizeCh
	}

	for {
		if err := u.update(); err != nil {
			return err
		}
	}
}

// outsideSize must be called on the same goroutine as update().
func (u *userInterfaceImpl) outsideSize() (float64, float64) {
	var outsideWidth, outsideHeight float64

	u.m.RLock()
	if u.gbuildWidthPx == 0 || u.gbuildHeightPx == 0 {
		outsideWidth = u.outsideWidth
		outsideHeight = u.outsideHeight
	} else {
		// gomobile build
		d := deviceScale()
		outsideWidth = float64(u.gbuildWidthPx) / d
		outsideHeight = float64(u.gbuildHeightPx) / d
	}
	u.m.RUnlock()

	return outsideWidth, outsideHeight
}

func (u *userInterfaceImpl) update() error {
	<-renderCh
	defer func() {
		renderEndCh <- struct{}{}
	}()

	w, h := u.outsideSize()
	if err := u.context.updateFrame(u.graphicsDriver, w, h, deviceScale()); err != nil {
		return err
	}
	return nil
}

func (u *userInterfaceImpl) ScreenSizeInFullscreen() (int, int) {
	// TODO: This function should return gbuildWidthPx, gbuildHeightPx,
	// but these values are not initialized until the main loop starts.
	return 0, 0
}

// SetOutsideSize is called from mobile/ebitenmobileview.
//
// SetOutsideSize is concurrent safe.
func (u *userInterfaceImpl) SetOutsideSize(outsideWidth, outsideHeight float64) {
	u.m.Lock()
	if u.outsideWidth != outsideWidth || u.outsideHeight != outsideHeight {
		u.outsideWidth = outsideWidth
		u.outsideHeight = outsideHeight
	}
	u.m.Unlock()
}

func (u *userInterfaceImpl) setGBuildSize(widthPx, heightPx int) {
	u.m.Lock()
	u.gbuildWidthPx = widthPx
	u.gbuildHeightPx = heightPx
	u.m.Unlock()

	u.once.Do(func() {
		close(u.setGBuildSizeCh)
	})
}

func (u *userInterfaceImpl) adjustPosition(x, y int) (int, int) {
	xf, yf := u.context.adjustPosition(float64(x), float64(y), deviceScale())
	return int(xf), int(yf)
}

func (u *userInterfaceImpl) CursorMode() CursorMode {
	return CursorModeHidden
}

func (u *userInterfaceImpl) SetCursorMode(mode CursorMode) {
	// Do nothing
}

func (u *userInterfaceImpl) CursorShape() CursorShape {
	return CursorShapeDefault
}

func (u *userInterfaceImpl) SetCursorShape(shape CursorShape) {
	// Do nothing
}

func (u *userInterfaceImpl) IsFullscreen() bool {
	return false
}

func (u *userInterfaceImpl) SetFullscreen(fullscreen bool) {
	// Do nothing
}

func (u *userInterfaceImpl) IsFocused() bool {
	return atomic.LoadInt32(&u.foreground) != 0
}

func (u *userInterfaceImpl) IsRunnableOnUnfocused() bool {
	return false
}

func (u *userInterfaceImpl) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	// Do nothing
}

func (u *userInterfaceImpl) SetFPSMode(mode FPSModeType) {
	u.fpsMode = mode
	u.updateExplicitRenderingModeIfNeeded()
}

func (u *userInterfaceImpl) updateExplicitRenderingModeIfNeeded() {
	if u.renderRequester == nil {
		return
	}
	u.renderRequester.SetExplicitRenderingMode(u.fpsMode == FPSModeVsyncOffMinimum)
}

func (u *userInterfaceImpl) DeviceScaleFactor() float64 {
	return deviceScale()
}

func (u *userInterfaceImpl) SetScreenTransparent(transparent bool) {
	// Do nothing
}

func (u *userInterfaceImpl) IsScreenTransparent() bool {
	return false
}

func (u *userInterfaceImpl) resetForTick() {
	u.input.resetForTick()
}

func (u *userInterfaceImpl) SetInitFocused(focused bool) {
	// Do nothing
}

func (u *userInterfaceImpl) Input() *Input {
	return &u.input
}

func (u *userInterfaceImpl) Window() Window {
	return &nullWindow{}
}

type Touch struct {
	ID TouchID
	X  int
	Y  int
}

func (u *userInterfaceImpl) UpdateInput(keys map[Key]struct{}, runes []rune, touches []Touch) {
	u.input.update(keys, runes, touches)
	if u.fpsMode == FPSModeVsyncOffMinimum {
		u.renderRequester.RequestRenderIfNeeded()
	}
}

type RenderRequester interface {
	SetExplicitRenderingMode(explicitRendering bool)
	RequestRenderIfNeeded()
}

func (u *userInterfaceImpl) SetRenderRequester(renderRequester RenderRequester) {
	u.renderRequester = renderRequester
	u.updateExplicitRenderingModeIfNeeded()
}

func (u *userInterfaceImpl) ScheduleFrame() {
	if u.renderRequester != nil && u.fpsMode == FPSModeVsyncOffMinimum {
		u.renderRequester.RequestRenderIfNeeded()
	}
}

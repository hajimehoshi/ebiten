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

// +build android ios

package mobile

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"unicode"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

var (
	glContextCh = make(chan gl.Context, 1)

	// renderCh receives when updating starts.
	renderCh = make(chan struct{})

	// renderEndCh receives when updating finishes.
	renderEndCh = make(chan struct{})

	theUI = &UserInterface{
		foreground: true,
		errCh:      make(chan error),

		// Give a default outside size so that the game can start without initializing them.
		outsideWidth:  640,
		outsideHeight: 480,
		sizeChanged:   true,
	}
)

func init() {
	theUI.input.ui = theUI
}

func Get() *UserInterface {
	return theUI
}

// Update is called from mobile/ebitenmobileview.
func (u *UserInterface) Update() error {
	select {
	case err := <-u.errCh:
		return err
	default:
	}

	if !u.IsFocused() {
		return nil
	}

	renderCh <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-renderEndCh
		if u.t != nil {
			// If there is a (main) thread, ensure that cancel is called after every other task is done.
			u.t.Call(func() error {
				cancel()
				return nil
			})
		} else {
			cancel()
		}
	}()

	if u.Graphics().IsGL() {
		if u.glWorker == nil {
			panic("mobile: glWorker must be initialized but not")
		}
		workAvailable := u.glWorker.WorkAvailable()
	loop:
		for {
			select {
			case <-workAvailable:
				u.glWorker.DoWork()
			case <-ctx.Done():
				break loop
			}
		}
		return nil
	} else {
		u.t.Loop(ctx)
	}
	return nil
}

type UserInterface struct {
	outsideWidth  float64
	outsideHeight float64

	sizeChanged bool
	foreground  bool
	errCh       chan error

	// Used for gomobile-build
	gbuildWidthPx   int
	gbuildHeightPx  int
	setGBuildSizeCh chan struct{}
	once            sync.Once

	context driver.UIContext

	input Input

	t        *thread.Thread
	glWorker gl.Worker

	m sync.RWMutex
}

func deviceScale() float64 {
	return devicescale.GetAt(0, 0)
}

// appMain is the main routine for gomobile-build mode.
func (u *UserInterface) appMain(a app.App) {
	var glctx gl.Context
	var sizeInited bool

	touches := map[touch.Sequence]*Touch{}
	keys := map[driver.Key]struct{}{}

	for e := range a.Events() {
		var updateInput bool
		var runes []rune

		switch e := a.Filter(e).(type) {
		case lifecycle.Event:
			switch e.Crosses(lifecycle.StageVisible) {
			case lifecycle.CrossOn:
				u.SetForeground(true)
				glctx, _ = e.DrawContext.(gl.Context)
				// Assume that glctx is always a same instance.
				// Then, only once initializing should be enough.
				if glContextCh != nil {
					glContextCh <- glctx
					glContextCh = nil
				}
				a.Send(paint.Event{})
			case lifecycle.CrossOff:
				u.SetForeground(false)
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
				touches[e.Sequence] = &Touch{
					ID: int(e.Sequence),
					X:  int(x),
					Y:  int(y),
				}
			case touch.TypeEnd:
				delete(touches, e.Sequence)
			}
			updateInput = true
		case key.Event:
			k, ok := gbuildKeyToDriverKey[e.Code]
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
			ts := []*Touch{}
			for _, t := range touches {
				ts = append(ts, t)
			}
			u.input.update(keys, runes, ts, nil)
		}
	}
}

func (u *UserInterface) SetForeground(foreground bool) {
	u.m.Lock()
	u.foreground = foreground
	u.m.Unlock()

	if foreground {
		hooks.ResumeAudio()
	} else {
		hooks.SuspendAudio()
	}
}

func (u *UserInterface) Run(context driver.UIContext) error {
	u.setGBuildSizeCh = make(chan struct{})
	go func() {
		if err := u.run(context, true); err != nil {
			// As mobile apps never ends, Loop can't return. Just panic here.
			panic(err)
		}
	}()
	app.Main(u.appMain)
	return nil
}

func (u *UserInterface) RunWithoutMainLoop(context driver.UIContext) {
	go func() {
		// title is ignored?
		if err := u.run(context, false); err != nil {
			u.errCh <- err
		}
	}()
}

func (u *UserInterface) run(context driver.UIContext, mainloop bool) (err error) {
	// Convert the panic to a regular error so that Java/Objective-C layer can treat this easily e.g., for
	// Crashlytics. A panic is treated as SIGABRT, and there is no way to handle this on Java/Objective-C layer
	// unfortunately.
	// TODO: Panic on other goroutines cannot be handled here.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v\n%q", r, string(debug.Stack()))
		}
	}()

	u.m.Lock()
	u.sizeChanged = true
	u.context = context
	u.m.Unlock()

	if u.Graphics().IsGL() {
		var ctx gl.Context
		if mainloop {
			ctx = <-glContextCh
		} else {
			ctx, u.glWorker = gl.NewContext()
		}
		u.Graphics().(*opengl.Driver).SetMobileGLContext(ctx)
	} else {
		u.t = thread.New()
		u.Graphics().SetThread(u.t)
	}

	// If gomobile-build is used, wait for the outside size fixed.
	if u.setGBuildSizeCh != nil {
		<-u.setGBuildSizeCh
	}

	// Force to set the screen size
	u.updateSize(context)
	for {
		if err := u.update(context); err != nil {
			return err
		}
	}
}

func (u *UserInterface) updateSize(context driver.UIContext) {
	var outsideWidth, outsideHeight float64

	u.m.Lock()
	sizeChanged := u.sizeChanged
	if sizeChanged {
		if u.gbuildWidthPx == 0 || u.gbuildHeightPx == 0 {
			outsideWidth = u.outsideWidth
			outsideHeight = u.outsideHeight
		} else {
			// gomobile build
			d := deviceScale()
			outsideWidth = float64(u.gbuildWidthPx) / d
			outsideHeight = float64(u.gbuildHeightPx) / d
		}
	}
	u.sizeChanged = false
	u.m.Unlock()

	if sizeChanged {
		context.Layout(outsideWidth, outsideHeight)
	}
}

func (u *UserInterface) update(context driver.UIContext) error {
	<-renderCh
	defer func() {
		renderEndCh <- struct{}{}
	}()

	if err := context.Update(func() {
		u.updateSize(context)
	}); err != nil {
		return err
	}
	return nil
}

func (u *UserInterface) ScreenSizeInFullscreen() (int, int) {
	// TODO: This function should return gbuildWidthPx, gbuildHeightPx,
	// but these values are not initialized until the main loop starts.
	return 0, 0
}

// SetOutsideSize is called from mobile/ebitenmobileview.
func (u *UserInterface) SetOutsideSize(outsideWidth, outsideHeight float64) {
	// Called from ebitenmobileview.
	u.m.Lock()
	if u.outsideWidth != outsideWidth || u.outsideHeight != outsideHeight {
		u.outsideWidth = outsideWidth
		u.outsideHeight = outsideHeight
		u.sizeChanged = true
	}
	u.m.Unlock()
}

func (u *UserInterface) setGBuildSize(widthPx, heightPx int) {
	u.m.Lock()
	u.gbuildWidthPx = widthPx
	u.gbuildHeightPx = heightPx
	u.sizeChanged = true
	u.once.Do(func() {
		close(u.setGBuildSizeCh)
	})
	u.m.Unlock()
}

func (u *UserInterface) adjustPosition(x, y int) (int, int) {
	xf, yf := u.context.AdjustPosition(float64(x), float64(y))
	return int(xf), int(yf)
}

func (u *UserInterface) CursorMode() driver.CursorMode {
	return driver.CursorModeHidden
}

func (u *UserInterface) SetCursorMode(mode driver.CursorMode) {
	// Do nothing
}

func (u *UserInterface) IsFullscreen() bool {
	return false
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	// Do nothing
}

func (u *UserInterface) IsFocused() bool {
	u.m.Lock()
	fg := u.foreground
	u.m.Unlock()
	return fg
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return false
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	// Do nothing
}

func (u *UserInterface) IsVsyncEnabled() bool {
	return true
}

func (u *UserInterface) SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func (u *UserInterface) DeviceScaleFactor() float64 {
	return deviceScale()
}

func (u *UserInterface) SetScreenTransparent(transparent bool) {
	// Do nothing
}

func (u *UserInterface) IsScreenTransparent() bool {
	return false
}

func (u *UserInterface) Input() driver.Input {
	return &u.input
}

func (u *UserInterface) Window() driver.Window {
	return nil
}

type Touch struct {
	ID int
	X  int
	Y  int
}

type Gamepad struct {
	ID        int
	SDLID     string
	Name      string
	Buttons   [driver.GamepadButtonNum]bool
	ButtonNum int
	Axes      [32]float32
	AxisNum   int
}

func (u *UserInterface) UpdateInput(keys map[driver.Key]struct{}, runes []rune, touches []*Touch, gamepads []Gamepad) {
	u.input.update(keys, runes, touches, gamepads)
}

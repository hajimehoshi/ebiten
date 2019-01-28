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

package ui

import (
	"errors"
	"image"
	"runtime"
	"sync"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/input"
)

var (
	glContextCh = make(chan gl.Context)
	renderCh    = make(chan struct{})
	renderChEnd = make(chan struct{})
	currentUI   = &userInterface{}
)

func Render(chError <-chan error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if chError == nil {
		return errors.New("ui: chError must not be nil")
	}
	// TODO: Check this is called on the rendering thread
	select {
	case err := <-chError:
		return err
	case renderCh <- struct{}{}:
		return opengl.Get().DoWork(renderChEnd)
	case <-time.After(500 * time.Millisecond):
		// This function must not be blocked. We need to break for timeout.
		return nil
	}
}

type userInterface struct {
	width       int
	height      int
	scale       float64
	sizeChanged bool

	// Used for gomobile-build
	fullscreenScale    float64
	fullscreenWidthPx  int
	fullscreenHeightPx int

	m sync.RWMutex
}

var (
	deviceScaleVal float64
	deviceScaleM   sync.Mutex
)

func getDeviceScale() float64 {
	deviceScaleM.Lock()
	defer deviceScaleM.Unlock()

	if deviceScaleVal == 0 {
		deviceScaleVal = devicescale.GetAt(0, 0)
	}
	return deviceScaleVal
}

// appMain is the main routine for gomobile-build mode.
func appMain(a app.App) {
	var glctx gl.Context
	touches := map[touch.Sequence]*input.Touch{}
	for e := range a.Events() {
		switch e := a.Filter(e).(type) {
		case lifecycle.Event:
			switch e.Crosses(lifecycle.StageVisible) {
			case lifecycle.CrossOn:
				glctx, _ = e.DrawContext.(gl.Context)
				// Assume that glctx is always a same instance.
				// Then, only once initializing should be enough.
				if glContextCh != nil {
					glContextCh <- glctx
					glContextCh = nil
				}
				a.Send(paint.Event{})
			case lifecycle.CrossOff:
				glctx = nil
			}
		case size.Event:
			setFullscreen(e.WidthPx, e.HeightPx)
		case paint.Event:
			if glctx == nil || e.External {
				continue
			}
			renderCh <- struct{}{}
			<-renderChEnd
			a.Publish()
			a.Send(paint.Event{})
		case touch.Event:
			switch e.Type {
			case touch.TypeBegin, touch.TypeMove:
				s := getDeviceScale()
				x, y := float64(e.X)/s, float64(e.Y)/s
				// TODO: Is it ok to cast from int64 to int here?
				t := input.NewTouch(int(e.Sequence), int(x), int(y))
				touches[e.Sequence] = t
			case touch.TypeEnd:
				delete(touches, e.Sequence)
			}
			ts := []*input.Touch{}
			for _, t := range touches {
				ts = append(ts, t)
			}
			UpdateTouches(ts)
		}
	}
}

func Run(width, height int, scale float64, title string, g GraphicsContext, mainloop bool) error {
	u := currentUI

	u.m.Lock()
	u.width = width
	u.height = height
	u.scale = scale
	u.sizeChanged = true
	u.m.Unlock()
	// title is ignored?

	if mainloop {
		ctx := <-glContextCh
		opengl.Get().InitWithContext(ctx)
	} else {
		opengl.Get().Init()
	}

	// Force to set the screen size
	u.updateGraphicsContext(g)
	for {
		if err := u.update(g); err != nil {
			return err
		}
	}
}

// Loop runs the main routine for gomobile-build.
func Loop(ch <-chan error) error {
	go func() {
		// As mobile apps never ends, Loop can't return. Just panic here.
		err := <-ch
		panic(err)
	}()
	app.Main(appMain)
	return nil
}

func (u *userInterface) updateGraphicsContext(g GraphicsContext) {
	width, height := 0, 0
	actualScale := 0.0

	u.m.Lock()
	sizeChanged := u.sizeChanged
	if sizeChanged {
		width = u.width
		height = u.height
		actualScale = u.scaleImpl() * getDeviceScale()
	}
	u.sizeChanged = false
	u.m.Unlock()

	if sizeChanged {
		// Sizing also calls GL functions
		g.SetSize(width, height, actualScale)
	}
}

func actualScale() float64 {
	return currentUI.actualScale()
}

func (u *userInterface) actualScale() float64 {
	u.m.Lock()
	s := u.scaleImpl() * getDeviceScale()
	u.m.Unlock()
	return s
}

func (u *userInterface) scaleImpl() float64 {
	scale := u.scale
	if u.fullscreenScale != 0 {
		scale = u.fullscreenScale
	}
	return scale
}

func (u *userInterface) update(g GraphicsContext) error {
render:
	for {
		select {
		case <-renderCh:
			break render
		case <-time.After(500 * time.Millisecond):
			hooks.SuspendAudio()
			continue
		}
	}
	hooks.ResumeAudio()

	defer func() {
		renderChEnd <- struct{}{}
	}()

	if err := g.Update(func() {
		u.updateGraphicsContext(g)
	}); err != nil {
		return err
	}
	return nil
}

func screenSize() (int, int) {
	return currentUI.screenSize()
}

func (u *userInterface) screenSize() (int, int) {
	u.m.Lock()
	w, h := u.width, u.height
	u.m.Unlock()
	return w, h
}

func ScreenSizeInFullscreen() (int, int) {
	// TODO: This function should return fullscreenWidthPx, fullscreenHeightPx,
	// but these values are not initialized until the main loop starts.
	return 0, 0
}

func SetScreenSize(width, height int) bool {
	currentUI.setScreenSize(width, height)
	return true
}

func (u *userInterface) setScreenSize(width, height int) {
	u.m.Lock()
	if u.width != width || u.height != height {
		u.width = width
		u.height = height
		u.updateFullscreenScaleIfNeeded()
		u.sizeChanged = true
	}
	u.m.Unlock()
}

func SetScreenScale(scale float64) bool {
	currentUI.setScreenScale(scale)
	return false
}

func (u *userInterface) setScreenScale(scale float64) {
	u.m.Lock()
	if u.scale != scale {
		u.scale = scale
		u.sizeChanged = true
	}
	u.m.Unlock()
}

func ScreenScale() float64 {
	u := currentUI
	u.m.RLock()
	s := u.scale
	u.m.RUnlock()
	return s
}

func setFullscreen(widthPx, heightPx int) {
	currentUI.setFullscreen(widthPx, heightPx)
}

func (u *userInterface) setFullscreen(widthPx, heightPx int) {
	u.m.Lock()
	u.fullscreenWidthPx = widthPx
	u.fullscreenHeightPx = heightPx
	u.updateFullscreenScaleIfNeeded()
	u.sizeChanged = true
	u.m.Unlock()
}

func (u *userInterface) updateFullscreenScaleIfNeeded() {
	if u.fullscreenWidthPx == 0 || u.fullscreenHeightPx == 0 {
		return
	}
	w, h := u.width, u.height
	scaleX := float64(u.fullscreenWidthPx) / float64(w)
	scaleY := float64(u.fullscreenHeightPx) / float64(h)
	scale := scaleX
	if scale > scaleY {
		scale = scaleY
	}
	u.fullscreenScale = scale / getDeviceScale()
	u.sizeChanged = true
}

func ScreenPadding() (x0, y0, x1, y1 float64) {
	return currentUI.screenPadding()
}

func (u *userInterface) screenPadding() (x0, y0, x1, y1 float64) {
	u.m.Lock()
	x0, y0, x1, y1 = u.screenPaddingImpl()
	u.m.Unlock()
	return
}

func (u *userInterface) screenPaddingImpl() (x0, y0, x1, y1 float64) {
	if u.fullscreenScale == 0 {
		return 0, 0, 0, 0
	}
	s := u.fullscreenScale * getDeviceScale()
	ox := (float64(u.fullscreenWidthPx) - float64(u.width)*s) / 2
	oy := (float64(u.fullscreenHeightPx) - float64(u.height)*s) / 2
	return ox, oy, ox, oy
}

func AdjustedCursorPosition() (x, y int) {
	return currentUI.adjustPosition(input.Get().CursorPosition())
}

func AdjustedTouches() []*input.Touch {
	ts := input.Get().Touches()
	adjusted := make([]*input.Touch, len(ts))
	for i, t := range ts {
		x, y := currentUI.adjustPosition(t.Position())
		adjusted[i] = input.NewTouch(t.ID(), x, y)
	}
	return adjusted
}

func (u *userInterface) adjustPosition(x, y int) (int, int) {
	u.m.Lock()
	ox, oy, _, _ := u.screenPaddingImpl()
	s := u.scaleImpl()
	as := s * getDeviceScale()
	u.m.Unlock()
	return int(float64(x)/s - ox/as), int(float64(y)/s - oy/as)
}

func IsCursorVisible() bool {
	return false
}

func SetCursorVisible(visible bool) {
	// Do nothing
}

func IsFullscreen() bool {
	return false
}

func SetFullscreen(fullscreen bool) {
	// Do nothing
}

func IsRunnableInBackground() bool {
	return false
}

func SetRunnableInBackground(runnableInBackground bool) {
	// Do nothing
}

func SetWindowTitle(title string) {
	// Do nothing
}

func SetWindowIcon(iconImages []image.Image) {
	// Do nothing
}

func IsWindowDecorated() bool {
	return false
}

func SetWindowDecorated(decorated bool) {
	// Do nothing
}

func IsWindowResizable() bool {
	return false
}

func SetWindowResizable(decorated bool) {
	// Do nothing
}

func IsVsyncEnabled() bool {
	return true
}

func SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func UpdateTouches(touches []*input.Touch) {
	input.Get().UpdateTouches(touches)
}

func DeviceScaleFactor() float64 {
	return getDeviceScale()
}

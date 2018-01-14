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

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func RunMainThreadLoop(ch <-chan error) error {
	return errors.New("ui: don't call this: use RunWithoutMainLoop instead of Run")
}

func Render(chError <-chan error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if chError == nil {
		return errors.New("ui: chError must not be nil")
	}
	// TODO: Check this is called on the rendering thread
	select {
	case chRender <- struct{}{}:
		return opengl.GetContext().DoWork(chError, chRenderEnd)
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

	m sync.RWMutex
}

var (
	chRender    = make(chan struct{})
	chRenderEnd = make(chan struct{})
	currentUI   = &userInterface{}
)

func Run(width, height int, scale float64, title string, g GraphicsContext) error {
	u := currentUI

	u.m.Lock()
	u.width = width
	u.height = height
	u.scale = scale
	u.sizeChanged = true
	u.m.Unlock()

	// title is ignored?
	opengl.Init()
	for {
		if err := u.update(g); err != nil {
			return err
		}
	}
}

func (u *userInterface) update(g GraphicsContext) error {
	<-chRender
	defer func() {
		chRenderEnd <- struct{}{}
	}()

	sizeChanged := false
	width, height := 0, 0
	actualScale := 0.0

	u.m.Lock()
	sizeChanged = u.sizeChanged
	if sizeChanged {
		width = u.width
		height = u.height
		actualScale = u.scale * devicescale.DeviceScale()
	}
	u.sizeChanged = false
	u.m.Unlock()

	if sizeChanged {
		// Sizing also calls GL functions
		g.SetSize(width, height, actualScale)
	}

	if err := g.Update(func() {}); err != nil {
		return err
	}
	return nil
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

func ScreenOffset() (float64, float64) {
	return 0, 0
}

func adjustCursorPosition(x, y int) (int, int) {
	return x, y
}

func IsCursorVisible() bool {
	return false
}

func SetCursorVisible(visible bool) {
	// Do nothing
}

func SetFullscreen(fullscreen bool) {
	// Do nothing
}

func IsFullscreen() bool {
	// Do nothing
	return false
}

func SetRunnableInBackground(runnableInBackground bool) {
	// Do nothing
}

func IsRunnableInBackground() bool {
	// Do nothing
	return false
}

func SetWindowIcon(iconImages []image.Image) {
	// Do nothing
}

func UpdateTouches(touches []Touch) {
	currentInput.updateTouches(touches)
}

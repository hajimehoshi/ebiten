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
	"time"

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
}

var (
	chRender    = make(chan struct{})
	chRenderEnd = make(chan struct{})
	currentUI   = &userInterface{
		sizeChanged: true,
	}
)

func Run(width, height int, scale float64, title string, g GraphicsContext) error {
	u := currentUI
	u.width = width
	u.height = height
	u.scale = scale
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

	if u.sizeChanged {
		// Sizing also calls GL functions
		u.sizeChanged = false
		g.SetSize(u.width, u.height, u.actualScreenScale())
		return nil
	}
	if err := g.Update(func() {}); err != nil {
		return err
	}
	return nil
}

func SetScreenSize(width, height int) bool {
	// TODO: Implement
	return false
}

func SetScreenScale(scale float64) bool {
	// TODO: Implement
	return false
}

func ScreenScale() float64 {
	return currentUI.scale
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

func SetCursorVisibility(visibility bool) {
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

func (u *userInterface) actualScreenScale() float64 {
	return u.scale * deviceScale()
}

func UpdateTouches(touches []Touch) {
	currentInput.updateTouches(touches)
}

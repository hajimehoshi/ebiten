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

// +build android

package ui

import (
	"errors"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

func initialize() (*opengl.Context, error) {
	return opengl.NewContext()
}

func Main() error {
	return errors.New("ui: don't call this: use RunWithoutMainLoop instead of Run")
}

func Render(chError <-chan error) error {
	if chError == nil {
		return errors.New("ui: chError must not be nil")
	}
	// TODO: Check this is called on the rendering thread
	chRender <- struct{}{}
	worker := glContext.Worker()
loop:
	for {
		select {
		case err := <-chError:
			return err
		case <-worker.WorkAvailable():
			worker.DoWork()
		case <-chRenderEnd:
			break loop
		}
	}
	return nil
}

type userInterface struct {
	width       int
	height      int
	scale       int
	sizeChanged bool
	render      chan struct{}
	renderEnd   chan struct{}
}

var (
	chRender    = make(chan struct{})
	chRenderEnd = make(chan struct{})
	currentUI   = &userInterface{
		sizeChanged: true,
		render:      chRender,
		renderEnd:   chRenderEnd,
	}
)

func CurrentUI() UserInterface {
	return currentUI
}

func (u *userInterface) Start(width, height, scale int, title string) error {
	u.width = width
	u.height = height
	u.scale = scale
	// title is ignored?
	return nil
}

func (u *userInterface) Terminate() error {
	return nil
}

func (u *userInterface) Update() (interface{}, error) {
	// TODO: Need lock?
	if u.sizeChanged {
		u.sizeChanged = false
		e := ScreenSizeEvent{
			Width:       u.width,
			Height:      u.height,
			Scale:       u.scale,
			ActualScale: u.actualScreenScale(),
		}
		return e, nil
	}
	select {
	case <-u.render:
		return RenderEvent{}, nil
	}
}

func (u *userInterface) SwapBuffers() error {
	return nil
}

func (u *userInterface) FinishRendering() error {
	u.renderEnd <- struct{}{}
	return nil
}

func (u *userInterface) SetScreenSize(width, height int) bool {
	// TODO: Implement
	return false
}

func (u *userInterface) SetScreenScale(scale int) bool {
	// TODO: Implement
	return false
}

func (u *userInterface) ScreenScale() int {
	return u.scale
}

func (u *userInterface) actualScreenScale() int {
	return u.scale
}

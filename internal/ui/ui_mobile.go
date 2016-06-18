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

// +build android ios darwin,arm darwin,arm64

package ui

import (
	"errors"
	"time"

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
	if chGLInitialized != nil {
		if err := doGLWorks(chError, glContext.InitializedCh()); err != nil {
			return err
		}
		close(chGLInitialized)
		<-chGLInitializedEnd
	}
	select {
	case chRender <- struct{}{}:
		return doGLWorks(chError, chRenderEnd)
	case <-time.After(500 * time.Millisecond):
		// This function must not be blocked. We need to break for timeout.
		return nil
	}
}

func doGLWorks(chError <-chan error, chDone <-chan struct{}) error {
	// TODO: Check this is called on the rendering thread
	worker := glContext.Worker()
loop:
	for {
		select {
		case err := <-chError:
			return err
		case <-worker.WorkAvailable():
			worker.DoWork()
		default:
			select {
			case <-chDone:
				break loop
			default:
			}
		}
	}
	return nil
}

type userInterface struct {
	width       int
	height      int
	scale       float64
	sizeChanged bool
}

var (
	chRender           = make(chan struct{})
	chRenderEnd        = make(chan struct{})
	chGLInitialized    = make(chan struct{})
	chGLInitializedEnd = make(chan struct{})
	currentUI          = &userInterface{
		sizeChanged: true,
	}
)

func CurrentUI() UserInterface {
	return currentUI
}

func (u *userInterface) Start(width, height int, scale float64, title string) error {
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
	if chGLInitialized != nil {
		<-chGLInitialized
		chGLInitialized = nil
		close(chGLInitializedEnd)
	}
	if u.sizeChanged {
		u.sizeChanged = false
		e := ScreenSizeEvent{
			Width:       u.width,
			Height:      u.height,
			ActualScale: u.actualScreenScale(),
		}
		return e, nil
	}
	<-chRender
	return RenderEvent{chRenderEnd}, nil
}

func (u *userInterface) SwapBuffers() error {
	return nil
}

func (u *userInterface) SetScreenSize(width, height int) bool {
	// TODO: Implement
	return false
}

func (u *userInterface) SetScreenScale(scale float64) bool {
	// TODO: Implement
	return false
}

func (u *userInterface) ScreenScale() float64 {
	return u.scale
}

func (u *userInterface) actualScreenScale() float64 {
	return u.scale * deviceScale()
}

func UpdateTouches(touches []Touch) {
	currentInput.updateTouches(touches)
}

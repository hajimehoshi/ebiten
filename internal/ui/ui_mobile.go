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
	select {
	case <-chPauseStart:
		if err := doGLWorks(chError, chPauseEnd); err != nil {
			return err
		}
		chPauseEnd2 <- struct{}{}
		return nil
	case <-chResumeStart:
		if err := doGLWorks(chError, chResumeEnd); err != nil {
			return err
		}
		return nil
	case chRender <- struct{}{}:
		return doGLWorks(chError, chRenderEnd)
	case <-time.After(500 * time.Millisecond):
		// This function must not be blocked so we need to break after a while.
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
	scale       int
	sizeChanged bool
	paused      bool
}

var (
	// TODO: Rename these channels
	chRender      = make(chan struct{})
	chRenderEnd   = make(chan struct{})
	chPause       = make(chan struct{})
	chPauseStart  = make(chan struct{})
	chPauseEnd    = make(chan struct{})
	chPauseEnd2   = make(chan struct{})
	chResume      = make(chan struct{})
	chResumeStart = make(chan struct{})
	chResumeEnd   = make(chan struct{})
	currentUI     = &userInterface{
		sizeChanged: true,
		paused:      false,
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
	if u.paused {
		select {
		case <-chResume:
			u.paused = false
			chResumeStart <- struct{}{}
			return ResumeEvent{chResumeEnd}, nil
		}
	}
	select {
	case <-chPause:
		u.paused = true
		chPauseStart <- struct{}{}
		return PauseEvent{chPauseEnd}, nil
	case <-chRender:
		return RenderEvent{chRenderEnd}, nil
	}
}

func (u *userInterface) SwapBuffers() error {
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

func Pause() error {
	// Pause must be done in the current GL context.
	chPause <- struct{}{}
	<-chPauseEnd2
	return nil
}

func Resume() error {
	chResume <- struct{}{}
	// Don't have to wait for resumeing done.
	return nil
}

func UpdateTouches(touches []Touch) {
	currentInput.updateTouches(touches)
}

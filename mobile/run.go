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

package mobile

import (
	"errors"
	"runtime"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

var chError <-chan error

type EventDispatcher interface {
	SetScreenSize(width, height int)
	SetScreenScale(scale int)
	Render() error
	Pause()
	Resume()
	TouchDown(x, y int)
	TouchUp(x, y int)
	TouchMove(x, y int)
}

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height, scale int, title string) (EventDispatcher, error) {
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, title)
	return &eventDispatcher{}, nil
}

type eventDispatcher struct {
}

func (e *eventDispatcher) SetScreenSize(width, height int) {
	ui.CurrentUI().SetScreenSize(width, height)
}

func (e *eventDispatcher) SetScreenScale(scale int) {
	ui.CurrentUI().SetScreenScale(scale)
}

func (e *eventDispatcher) Render() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if chError == nil {
		return errors.New("mobile: chError must not be nil: Start is not called yet?")
	}
	return ui.Render(chError)
}

func (e *eventDispatcher) Pause() {
	ui.Pause()
}

func (e *eventDispatcher) Resume() {
	ui.Resume()
}

func (e *eventDispatcher) TouchDown(x, y int) {
	ui.TouchDown(x, y)
}

func (e *eventDispatcher) TouchUp(x, y int) {
	ui.TouchUp(x, y)
}

func (e *eventDispatcher) TouchMove(x, y int) {
	ui.TouchMove(x, y)
}

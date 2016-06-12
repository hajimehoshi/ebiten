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

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

var chError <-chan error

type EventDispatcher interface {
	SetScreenSize(width, height int)
	SetScreenScale(scale int)
	Render() error
	TouchDown(id int, x, y int)
	TouchMove(id int, x, y int)
	TouchUp(id int)
}

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height, scale int, title string) (EventDispatcher, error) {
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, title)
	return &eventDispatcher{
		touches: map[int]position{},
	}, nil
}

type position struct {
	x int
	y int
}

type eventDispatcher struct {
	touches map[int]position
}

func (e *eventDispatcher) SetScreenSize(width, height int) {
	ui.CurrentUI().SetScreenSize(width, height)
}

func (e *eventDispatcher) SetScreenScale(scale int) {
	ui.CurrentUI().SetScreenScale(scale)
}

func (e *eventDispatcher) Render() error {
	if chError == nil {
		return errors.New("mobile: chError must not be nil: Start is not called yet?")
	}
	return ui.Render(chError)
}

// touch implements ui.Touch.
type touch struct {
	id       int
	position position
}

func (t touch) ID() int {
	return t.id
}

func (t touch) Position() (int, int) {
	// TODO: Is this OK to adjust the position here?
	return t.position.x / ui.CurrentUI().ScreenScale(),
		t.position.y / ui.CurrentUI().ScreenScale()
}

func (e *eventDispatcher) TouchDown(id int, x, y int) {
	e.touches[id] = position{x, y}
	e.updateTouches()
}

func (e *eventDispatcher) TouchMove(id int, x, y int) {
	e.touches[id] = position{x, y}
	e.updateTouches()
}

func (e *eventDispatcher) TouchUp(id int) {
	delete(e.touches, id)
	e.updateTouches()
}

func (e *eventDispatcher) updateTouches() {
	ts := []ui.Touch{}
	for id, position := range e.touches {
		ts = append(ts, touch{id, position})
	}
	ui.UpdateTouches(ts)
}

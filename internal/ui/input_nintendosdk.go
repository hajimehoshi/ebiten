// Copyright 2021 The Ebiten Authors
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

//go:build nintendosdk
// +build nintendosdk

package ui

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/nintendosdk"
)

type Input struct {
	gamepads []nintendosdk.Gamepad
	touches  []nintendosdk.Touch

	m sync.Mutex
}

func (i *Input) update(context *context) {
	i.m.Lock()
	defer i.m.Unlock()

	gamepad.Update()

	i.touches = i.touches[:0]
	i.touches = nintendosdk.AppendTouches(i.touches)

	for idx, t := range i.touches {
		x, y := context.adjustPosition(float64(t.X), float64(t.Y), deviceScaleFactor)
		i.touches[idx].X = int(x)
		i.touches[idx].Y = int(y)
	}
}

func (i *Input) AppendInputChars(runes []rune) []rune {
	return nil
}

func (i *Input) AppendTouchIDs(touchIDs []TouchID) []TouchID {
	i.m.Lock()
	defer i.m.Unlock()

	for _, t := range i.touches {
		touchIDs = append(touchIDs, TouchID(t.ID))
	}
	return touchIDs
}

func (i *Input) CursorPosition() (x, y int) {
	return 0, 0
}

func (i *Input) IsKeyPressed(key Key) bool {
	return false
}

func (i *Input) IsMouseButtonPressed(button MouseButton) bool {
	return false
}

func (i *Input) TouchPosition(id TouchID) (x, y int) {
	i.m.Lock()
	defer i.m.Unlock()

	for _, t := range i.touches {
		if TouchID(t.ID) == id {
			return t.X, t.Y
		}
	}
	return 0, 0
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}

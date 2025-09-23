// Copyright 2025 The Ebitengine Authors
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

// Package inputstate provides APIs to access the input state for the current tick.
// This package is for the ebiten package and the inpututil package.
package inputstate

import (
	"io/fs"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var (
	theInputState inputState
)

type inputState struct {
	state ui.InputState
	m     sync.Mutex
}

func Get() *inputState {
	return &theInputState
}

func (i *inputState) Update(fn func(*ui.InputState)) {
	i.m.Lock()
	defer i.m.Unlock()
	fn(&i.state)
}

func (i *inputState) AppendInputChars(runes []rune) []rune {
	i.m.Lock()
	defer i.m.Unlock()
	return append(runes, i.state.Runes...)
}

func (i *inputState) IsKeyPressed(key ui.Key) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsKeyPressed(key, ui.Get().Tick())
}

func (i *inputState) IsKeyJustPressed(key ui.Key) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsKeyJustPressed(key, ui.Get().Tick())
}

func (i *inputState) IsKeyJustReleased(key ui.Key) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsKeyJustReleased(key, ui.Get().Tick())
}

func (i *inputState) KeyPressDuration(key ui.Key) int64 {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.KeyPressDuration(key, ui.Get().Tick())
}

func AppendPressedKeys[T ~int](keys []T) []T {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	tick := ui.Get().Tick()
	for k := ui.Key(0); k <= ui.KeyMax; k++ {
		if !theInputState.state.IsKeyPressed(k, tick) {
			continue
		}
		keys = append(keys, T(k))
	}
	return keys
}

func AppendJustPressedKeys[T ~int](keys []T) []T {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	tick := ui.Get().Tick()
	for k := ui.Key(0); k <= ui.KeyMax; k++ {
		if !theInputState.state.IsKeyJustPressed(k, tick) {
			continue
		}
		keys = append(keys, T(k))
	}
	return keys
}

func AppendJustReleasedKeys[T ~int](keys []T) []T {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	tick := ui.Get().Tick()
	for k := ui.Key(0); k <= ui.KeyMax; k++ {
		if !theInputState.state.IsKeyJustReleased(k, tick) {
			continue
		}
		keys = append(keys, T(k))
	}
	return keys
}

func (i *inputState) IsMouseButtonPressed(mouseButton ui.MouseButton) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsMouseButtonPressed(mouseButton, ui.Get().Tick())
}

func (i *inputState) IsMouseButtonJustPressed(mouseButton ui.MouseButton) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsMouseButtonJustPressed(mouseButton, ui.Get().Tick())
}

func (i *inputState) IsMouseButtonJustReleased(mouseButton ui.MouseButton) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.IsMouseButtonJustReleased(mouseButton, ui.Get().Tick())
}

func (i *inputState) MouseButtonPressDuration(mouseButton ui.MouseButton) int64 {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.MouseButtonPressDuration(mouseButton, ui.Get().Tick())
}

func (i *inputState) CursorPosition() (float64, float64) {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.CursorX, i.state.CursorY
}

func (i *inputState) Wheel() (float64, float64) {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.WheelX, i.state.WheelY
}

func AppendTouchIDs[T ~int](touches []T) []T {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	for _, t := range theInputState.state.Touches {
		touches = append(touches, T(t.ID))
	}
	return touches
}

func (i *inputState) TouchPosition(id ui.TouchID) (int, int) {
	i.m.Lock()
	defer i.m.Unlock()

	for _, t := range i.state.Touches {
		if id != ui.TouchID(t.ID) {
			continue
		}
		return t.X, t.Y
	}
	return 0, 0
}

func (i *inputState) WindowBeingClosed() bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.WindowBeingClosed
}

func (i *inputState) DroppedFiles() fs.FS {
	i.m.Lock()
	defer i.m.Unlock()
	return i.state.DroppedFiles
}

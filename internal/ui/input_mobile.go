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

//go:build (android || ios) && !nintendosdk
// +build android ios
// +build !nintendosdk

package ui

type Input struct {
	keys    map[Key]struct{}
	runes   []rune
	touches []Touch
	ui      *userInterfaceImpl
}

func (i *Input) CursorPosition() (x, y int) {
	return 0, 0
}

func (i *Input) AppendTouchIDs(touchIDs []TouchID) []TouchID {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, t := range i.touches {
		touchIDs = append(touchIDs, t.ID)
	}
	return touchIDs
}

func (i *Input) TouchPosition(id TouchID) (x, y int) {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, t := range i.touches {
		if t.ID == id {
			return i.ui.adjustPosition(t.X, t.Y)
		}
	}
	return 0, 0
}

func (i *Input) AppendInputChars(runes []rune) []rune {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()
	return append(runes, i.runes...)
}

func (i *Input) IsKeyPressed(key Key) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	_, ok := i.keys[key]
	return ok
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}

func (i *Input) IsMouseButtonPressed(key MouseButton) bool {
	return false
}

func (i *Input) update(keys map[Key]struct{}, runes []rune, touches []Touch) {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if i.keys == nil {
		i.keys = map[Key]struct{}{}
	}
	for k := range i.keys {
		delete(i.keys, k)
	}
	for k := range keys {
		i.keys[k] = struct{}{}
	}

	i.runes = i.runes[:0]
	i.runes = append(i.runes, runes...)

	i.touches = i.touches[:0]
	i.touches = append(i.touches, touches...)
}

func (i *Input) resetForTick() {
	i.runes = nil
}

// Copyright 2015 Hajime Hoshi
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

package input

var theInput = &Input{}

func Get() *Input {
	return theInput
}

func (i *Input) CursorPosition() (x, y int) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.cursorX, i.cursorY
}

var emptyIDs = []int{}

func (i *Input) GamepadIDs() []int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) == 0 {
		// Avoid creating a slice if possible.
		// This is a performance optimization for browsers.
		return emptyIDs
	}
	r := []int{}
	for id, g := range i.gamepads {
		if g.valid {
			r = append(r, id)
		}
	}
	return r
}

func (i *Input) GamepadAxisNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *Input) GamepadAxis(id int, axis int) float64 {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *Input) GamepadButtonNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *Input) IsGamepadButtonPressed(id int, button GamepadButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}

var emptyTouches = []*Touch{}

func (in *Input) Touches() []*Touch {
	in.m.RLock()
	defer in.m.RUnlock()

	if len(in.touches) == 0 {
		// Avoid creating a slice if possible.
		// This is a performance optimization for browsers.
		return emptyTouches
	}

	t := make([]*Touch, len(in.touches))
	for i := 0; i < len(t); i++ {
		t[i] = in.touches[i]
	}
	return t
}

type gamePad struct {
	valid         bool
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

type Touch struct {
	id int
	x  int
	y  int
}

func NewTouch(id int, x, y int) *Touch {
	return &Touch{
		id: id,
		x:  x,
		y:  y,
	}
}

func (t *Touch) ID() int {
	return t.id
}

func (t *Touch) Position() (x, y int) {
	return t.x, t.y
}

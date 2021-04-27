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

package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

type pos struct {
	X int
	Y int
}

type Input struct {
	keys     map[driver.Key]struct{}
	runes    []rune
	touches  map[driver.TouchID]pos
	gamepads []Gamepad
	ui       *UserInterface
}

func (i *Input) CursorPosition() (x, y int) {
	return 0, 0
}

func (i *Input) GamepadIDs() []driver.GamepadID {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	ids := make([]driver.GamepadID, 0, len(i.gamepads))
	for _, g := range i.gamepads {
		ids = append(ids, g.ID)
	}
	return ids
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.SDLID
	}
	return ""
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.Name
	}
	return ""
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.AxisNum
	}
	return 0
}

func (i *Input) GamepadAxis(id driver.GamepadID, axis int) float64 {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if g.AxisNum <= int(axis) {
			return 0
		}
		return float64(g.Axes[axis])
	}
	return 0
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.ButtonNum
	}
	return 0
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if g.ButtonNum <= int(button) {
			return false
		}
		return g.Buttons[button]
	}
	return false
}

func (i *Input) TouchIDs() []driver.TouchID {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	if len(i.touches) == 0 {
		return nil
	}

	ids := make([]driver.TouchID, 0, len(i.touches))
	for id := range i.touches {
		ids = append(ids, id)
	}
	return ids
}

func (i *Input) TouchPosition(id driver.TouchID) (x, y int) {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for tid, pos := range i.touches {
		if id == tid {
			return i.ui.adjustPosition(pos.X, pos.Y)
		}
	}
	return 0, 0
}

func (i *Input) RuneBuffer() []rune {
	rs := make([]rune, len(i.runes))
	copy(rs, i.runes)
	return rs
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	if i.keys == nil {
		return false
	}
	_, ok := i.keys[key]
	return ok
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}

func (i *Input) IsMouseButtonPressed(key driver.MouseButton) bool {
	return false
}

func (i *Input) update(keys map[driver.Key]struct{}, runes []rune, touches []*Touch, gamepads []Gamepad) {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	i.keys = map[driver.Key]struct{}{}
	for k := range keys {
		i.keys[k] = struct{}{}
	}

	i.runes = make([]rune, len(runes))
	copy(i.runes, runes)

	i.touches = map[driver.TouchID]pos{}
	for _, t := range touches {
		i.touches[t.ID] = pos{
			X: t.X,
			Y: t.Y,
		}
	}

	i.gamepads = make([]Gamepad, len(gamepads))
	copy(i.gamepads, gamepads)
}

func (i *Input) resetForFrame() {
	i.runes = nil
}

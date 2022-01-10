// Copyright 2022 The Ebiten Authors
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

package gamepad

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

const (
	hatCentered  = 0
	hatUp        = 1
	hatRight     = 2
	hatDown      = 4
	hatLeft      = 8
	hatRightUp   = hatRight | hatUp
	hatRightDown = hatRight | hatDown
	hatLeftUp    = hatLeft | hatUp
	hatLeftDown  = hatLeft | hatDown
)

type gamepads struct {
	gamepads []*Gamepad
	m        sync.Mutex

	nativeGamepads
}

var theGamepads gamepads

func init() {
	theGamepads.nativeGamepads.init()
}

func AppendGamepadIDs(ids []driver.GamepadID) []driver.GamepadID {
	return theGamepads.appendGamepadIDs(ids)
}

func Update() {
	theGamepads.update()
}

func Get(id driver.GamepadID) *Gamepad {
	return theGamepads.get(id)
}

func (g *gamepads) appendGamepadIDs(ids []driver.GamepadID) []driver.GamepadID {
	g.m.Lock()
	defer g.m.Unlock()

	for i, gp := range g.gamepads {
		if gp != nil && gp.present() {
			ids = append(ids, driver.GamepadID(i))
		}
	}
	return ids
}

func (g *gamepads) update() {
	g.m.Lock()
	defer g.m.Unlock()

	for _, gp := range g.gamepads {
		if gp != nil {
			gp.update()
		}
	}
}

func (g *gamepads) get(id driver.GamepadID) *Gamepad {
	g.m.Lock()
	defer g.m.Unlock()

	if id < 0 || int(id) >= len(g.gamepads) {
		return nil
	}
	return g.gamepads[id]
}

func (g *gamepads) find(cond func(*Gamepad) bool) *Gamepad {
	g.m.Lock()
	defer g.m.Unlock()

	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if cond(gp) {
			return gp
		}
	}
	return nil
}

func (g *gamepads) add(name, sdlID string) *Gamepad {
	g.m.Lock()
	defer g.m.Unlock()

	for i, gp := range g.gamepads {
		if gp == nil {
			gp := &Gamepad{
				name:  name,
				sdlID: sdlID,
			}
			g.gamepads[i] = gp
			return gp
		}
	}

	gp := &Gamepad{
		name:  name,
		sdlID: sdlID,
	}
	g.gamepads = append(g.gamepads, gp)
	return gp
}

func (g *gamepads) remove(cond func(*Gamepad) bool) {
	g.m.Lock()
	defer g.m.Unlock()

	for i, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if cond(gp) {
			g.gamepads[i] = nil
		}
	}
}

type Gamepad struct {
	name  string
	sdlID string
	m     sync.Mutex

	nativeGamepad
}

func (g *Gamepad) update() {
	g.m.Lock()
	defer g.m.Unlock()

	g.nativeGamepad.update()
}

func (g *Gamepad) Name() string {
	// This is immutable and doesn't have to be protected by a mutex.
	return g.name
}

func (g *Gamepad) SDLID() string {
	// This is immutable and doesn't have to be protected by a mutex.
	return g.sdlID
}

func (g *Gamepad) AxisNum() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.axisNum()
}

func (g *Gamepad) ButtonNum() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.buttonNum()
}

func (g *Gamepad) HatNum() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.hatNum()
}

func (g *Gamepad) Axis(axis int) float64 {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.axisValue(axis)
}

func (g *Gamepad) Button(button int) bool {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.isButtonPressed(button)
}

func (g *Gamepad) Hat(hat int) int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.nativeGamepad.hatState(hat)
}

func (g *Gamepad) IsStandardLayoutAvailable() bool {
	g.m.Lock()
	defer g.m.Unlock()

	return gamepaddb.HasStandardLayoutMapping(g.sdlID)
}

func (g *Gamepad) StandardAxisValue(axis driver.StandardGamepadAxis) float64 {
	return gamepaddb.AxisValue(g.sdlID, axis, g)
}

func (g *Gamepad) StandardButtonValue(button driver.StandardGamepadButton) float64 {
	return gamepaddb.ButtonValue(g.sdlID, button, g)
}

func (g *Gamepad) IsStandardButtonPressed(button driver.StandardGamepadButton) bool {
	return gamepaddb.IsButtonPressed(g.sdlID, button, g)
}

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

//go:build (darwin && !ios) || js
// +build darwin,!ios js

package gamepad

import (
	"sync"
	"time"

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

	nativeGamepads
}

var theGamepads gamepads

func init() {
	theGamepads.nativeGamepads.init()
	theGamepads.nativeGamepads.gamepads = &theGamepads
}

// AppendGamepadIDs must be called on the main thread.
func AppendGamepadIDs(ids []driver.GamepadID) []driver.GamepadID {
	return theGamepads.appendGamepadIDs(ids)
}

// Update must be called on the main thread.
func Update() {
	theGamepads.update()
}

// Get must be called on the main thread.
func Get(id driver.GamepadID) *Gamepad {
	return theGamepads.get(id)
}

func (g *gamepads) appendGamepadIDs(ids []driver.GamepadID) []driver.GamepadID {
	for i, gp := range g.gamepads {
		if gp != nil && gp.present() {
			ids = append(ids, driver.GamepadID(i))
		}
	}
	return ids
}

func (g *gamepads) update() {
	g.nativeGamepads.update()
	for _, gp := range g.gamepads {
		if gp != nil {
			gp.update()
		}
	}
}

func (g *gamepads) get(id driver.GamepadID) *Gamepad {
	if id < 0 || int(id) >= len(g.gamepads) {
		return nil
	}
	return g.gamepads[id]
}

// find can be invoked from callbacks on the OS's main thread.
// As a callback can be called synchronously from update, using a mutex is not a good idea.
func (g *gamepads) find(cond func(*Gamepad) bool) *Gamepad {
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

// add can be invoked from callbacks on the OS's main thread.
// As a callback can be called synchronously from update, using a mutex is not a good idea.
func (g *gamepads) add(name, sdlID string) *Gamepad {
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

// remove can be invoked from callbacks on the OS's main thread.
// As a callback can be called synchronously from update, using a mutex is not a good idea.
func (g *gamepads) remove(cond func(*Gamepad) bool) {
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
	if name := gamepaddb.Name(g.sdlID); name != "" {
		return name
	}
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

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return true
	}
	return g.hasOwnStandardLayoutMapping()
}

func (g *Gamepad) StandardAxisValue(axis driver.StandardGamepadAxis) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.AxisValue(g.sdlID, axis, g)
	}
	if g.hasOwnStandardLayoutMapping() {
		return g.nativeGamepad.axisValue(int(axis))
	}
	return 0
}

func (g *Gamepad) StandardButtonValue(button driver.StandardGamepadButton) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.ButtonValue(g.sdlID, button, g)
	}
	if g.hasOwnStandardLayoutMapping() {
		return g.nativeGamepad.buttonValue(int(button))
	}
	return 0
}

func (g *Gamepad) IsStandardButtonPressed(button driver.StandardGamepadButton) bool {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.IsButtonPressed(g.sdlID, button, g)
	}
	if g.hasOwnStandardLayoutMapping() {
		return g.nativeGamepad.isButtonPressed(int(button))
	}
	return false
}

func (g *Gamepad) Vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g.nativeGamepad.vibrate(duration, strongMagnitude, weakMagnitude)
}

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
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type ID int

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
	inited   bool
	gamepads []*Gamepad
	m        sync.Mutex

	native nativeGamepads
}

type nativeGamepads interface {
	init(gamepads *gamepads) error
	update(gamepads *gamepads) error
}

var theGamepads = gamepads{
	native: newNativeGamepadsImpl(),
}

// AppendGamepadIDs is concurrent-safe.
func AppendGamepadIDs(ids []ID) []ID {
	return theGamepads.appendGamepadIDs(ids)
}

// Update is concurrent-safe.
func Update() error {
	return theGamepads.update()
}

// Get is concurrent-safe.
func Get(id ID) *Gamepad {
	return theGamepads.get(id)
}

func SetNativeWindow(nativeWindow uintptr) {
	theGamepads.setNativeWindow(nativeWindow)
}

func (g *gamepads) appendGamepadIDs(ids []ID) []ID {
	g.m.Lock()
	defer g.m.Unlock()

	for i, gp := range g.gamepads {
		if gp != nil {
			ids = append(ids, ID(i))
		}
	}
	return ids
}

func (g *gamepads) update() error {
	g.m.Lock()
	defer g.m.Unlock()

	if !g.inited {
		if err := g.native.init(g); err != nil {
			return err
		}
		g.inited = true
	}

	if err := g.native.update(g); err != nil {
		return err
	}

	// A gamepad can be detected even though there are not. Apparently, some special devices are
	// recognized as gamepads by OSes. In this case, the number of the 'buttons' can exceeds the
	// maximum. Skip such devices as a tentative solution (#1173, #2039).
	g.remove(func(gamepad *Gamepad) bool {
		return gamepad.ButtonCount() > ButtonNum
	})

	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if err := gp.update(g); err != nil {
			return err
		}
	}
	return nil
}

func (g *gamepads) get(id ID) *Gamepad {
	g.m.Lock()
	defer g.m.Unlock()

	if id < 0 || int(id) >= len(g.gamepads) {
		return nil
	}
	return g.gamepads[id]
}

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

func (g *gamepads) setNativeWindow(nativeWindow uintptr) {
	g.m.Lock()
	defer g.m.Unlock()

	var n interface{} = g.native
	if n, ok := n.(interface{ setNativeWindow(uintptr) }); ok {
		n.setNativeWindow(nativeWindow)
	}
}

type Gamepad struct {
	name  string
	sdlID string
	m     sync.Mutex

	native nativeGamepad
}

type nativeGamepad interface {
	update(gamepads *gamepads) error
	hasOwnStandardLayoutMapping() bool
	isStandardAxisAvailableInOwnMapping(axis gamepaddb.StandardAxis) bool
	isStandardButtonAvailableInOwnMapping(button gamepaddb.StandardButton) bool
	axisCount() int
	buttonCount() int
	hatCount() int
	axisValue(axis int) float64
	buttonValue(button int) float64
	isButtonPressed(button int) bool
	hatState(hat int) int
	vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64)
}

func (g *Gamepad) update(gamepads *gamepads) error {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.update(gamepads)
}

// Name is concurrent-safe.
func (g *Gamepad) Name() string {
	// This is immutable and doesn't have to be protected by a mutex.
	if name := gamepaddb.Name(g.sdlID); name != "" {
		return name
	}
	return g.name
}

// SDLID is concurrent-safe.
func (g *Gamepad) SDLID() string {
	// This is immutable and doesn't have to be protected by a mutex.
	return g.sdlID
}

// AxisCount is concurrent-safe.
func (g *Gamepad) AxisCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.axisCount()
}

// ButtonCount is concurrent-safe.
func (g *Gamepad) ButtonCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.buttonCount()
}

// HatCount is concurrent-safe.
func (g *Gamepad) HatCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.hatCount()
}

// Axis is concurrent-safe.
func (g *Gamepad) Axis(axis int) float64 {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.axisValue(axis)
}

// Button is concurrent-safe.
func (g *Gamepad) Button(button int) bool {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.isButtonPressed(button)
}

// Hat is concurrent-safe.
func (g *Gamepad) Hat(hat int) int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.hatState(hat)
}

// IsStandardLayoutAvailable is concurrent-safe.
func (g *Gamepad) IsStandardLayoutAvailable() bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return true
	}
	return g.native.hasOwnStandardLayoutMapping()
}

// IsStandardAxisAvailable is concurrent safe.
func (g *Gamepad) IsStandardAxisAvailable(button gamepaddb.StandardAxis) bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.HasStandardAxis(g.sdlID, button)
	}
	return g.native.isStandardAxisAvailableInOwnMapping(button)
}

// IsStandardButtonAvailable is concurrent safe.
func (g *Gamepad) IsStandardButtonAvailable(button gamepaddb.StandardButton) bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.HasStandardButton(g.sdlID, button)
	}
	return g.native.isStandardButtonAvailableInOwnMapping(button)
}

// StandardAxisValue is concurrent-safe.
func (g *Gamepad) StandardAxisValue(axis gamepaddb.StandardAxis) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.AxisValue(g.sdlID, axis, g)
	}
	if g.native.hasOwnStandardLayoutMapping() {
		return g.native.axisValue(int(axis))
	}
	return 0
}

// StandardButtonValue is concurrent-safe.
func (g *Gamepad) StandardButtonValue(button gamepaddb.StandardButton) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.ButtonValue(g.sdlID, button, g)
	}
	if g.native.hasOwnStandardLayoutMapping() {
		return g.native.buttonValue(int(button))
	}
	return 0
}

// IsStandardButtonPressed is concurrent-safe.
func (g *Gamepad) IsStandardButtonPressed(button gamepaddb.StandardButton) bool {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.IsButtonPressed(g.sdlID, button, g)
	}
	if g.native.hasOwnStandardLayoutMapping() {
		return g.native.isButtonPressed(int(button))
	}
	return false
}

// Vibrate is concurrent-safe.
func (g *Gamepad) Vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g.m.Lock()
	defer g.m.Unlock()

	g.native.vibrate(duration, strongMagnitude, weakMagnitude)
}

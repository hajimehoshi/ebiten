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

//go:build nintendosdk
// +build nintendosdk

package gamepad

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/nintendosdk"
)

type nativeGamepadsImpl struct {
	gamepads []nintendosdk.Gamepad
	ids      map[int]struct{}
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (*nativeGamepadsImpl) init(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	g.gamepads = g.gamepads[:0]
	g.gamepads = nintendosdk.AppendGamepads(g.gamepads)

	for id := range g.ids {
		delete(g.ids, id)
	}

	for _, gp := range g.gamepads {
		if g.ids == nil {
			g.ids = map[int]struct{}{}
		}
		g.ids[gp.ID] = struct{}{}

		gamepad := gamepads.find(func(gamepad *Gamepad) bool {
			return gamepad.native.(*nativeGamepadImpl).id == gp.ID
		})
		if gamepad == nil {
			gamepad = gamepads.add("", "")
			gamepad.native = &nativeGamepadImpl{
				id:            gp.ID,
				standard:      gp.Standard,
				axisValues:    make([]float64, gp.AxisCount),
				buttonPressed: make([]bool, gp.ButtonCount),
				buttonValues:  make([]float64, gp.ButtonCount),
			}
		}

		gamepad.m.Lock()
		n := gamepad.native.(*nativeGamepadImpl)
		copy(n.axisValues, gp.AxisValues[:])
		copy(n.buttonValues, gp.ButtonValues[:])
		copy(n.buttonPressed, gp.ButtonPressed[:])
		gamepad.m.Unlock()
	}

	// Remove an unused gamepads.
	gamepads.remove(func(gamepad *Gamepad) bool {
		_, ok := g.ids[gamepad.native.(*nativeGamepadImpl).id]
		return !ok
	})

	return nil
}

type nativeGamepadImpl struct {
	id       int
	standard bool

	axisValues    []float64
	buttonPressed []bool
	buttonValues  []float64
}

func (*nativeGamepadImpl) update(gamepad *gamepads) error {
	return nil
}

func (g *nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return g.standard
}

func (g *nativeGamepadImpl) isStandardAxisAvailableInOwnMapping(axis gamepaddb.StandardAxis) bool {
	// TODO: Implement this on the C side.
	return axis >= 0 && int(axis) < len(g.axisValues)
}

func (g *nativeGamepadImpl) isStandardButtonAvailableInOwnMapping(button gamepaddb.StandardButton) bool {
	// TODO: Implement this on the C side.
	return button >= 0 && int(button) < len(g.buttonValues)
}

func (g *nativeGamepadImpl) axisCount() int {
	return len(g.axisValues)
}

func (g *nativeGamepadImpl) buttonCount() int {
	return len(g.buttonValues)
}

func (g *nativeGamepadImpl) hatCount() int {
	return 0
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axisValues) {
		return 0
	}
	return g.axisValues[axis]
}

func (g *nativeGamepadImpl) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttonPressed) {
		return false
	}
	return g.buttonPressed[button]
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	if button < 0 || button >= len(g.buttonValues) {
		return 0
	}
	return g.buttonValues[button]
}

func (*nativeGamepadImpl) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	nintendosdk.VibrateGamepad(g.id, duration, strongMagnitude, weakMagnitude)
}

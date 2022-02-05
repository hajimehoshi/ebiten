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

//go:build ebitencbackend
// +build ebitencbackend

package gamepad

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/cbackend"
)

type nativeGamepads struct {
	gamepads []cbackend.Gamepad
	ids      map[int]struct{}
}

func (*nativeGamepads) init(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepads) update(gamepads *gamepads) error {
	g.gamepads = g.gamepads[:0]
	g.gamepads = cbackend.AppendGamepads(g.gamepads)

	for id := range g.ids {
		delete(g.ids, id)
	}

	for _, gp := range g.gamepads {
		if g.ids == nil {
			g.ids = map[int]struct{}{}
		}
		g.ids[gp.ID] = struct{}{}

		gamepad := gamepads.find(func(gamepad *Gamepad) bool {
			return gamepad.id == gp.ID
		})
		if gamepad == nil {
			gamepad = gamepads.add("", "")
			gamepad.id = gp.ID
			gamepad.standard = gp.Standard
			gamepad.axisValues = make([]float64, gp.AxisCount)
			gamepad.buttonPressed = make([]bool, gp.ButtonCount)
			gamepad.buttonValues = make([]float64, gp.ButtonCount)
		}

		gamepad.m.Lock()
		copy(gamepad.axisValues, gp.AxisValues[:])
		copy(gamepad.buttonValues, gp.ButtonValues[:])
		copy(gamepad.buttonPressed, gp.ButtonPressed[:])
		gamepad.m.Unlock()
	}

	// Remove an unused gamepads.
	gamepads.remove(func(gamepad *Gamepad) bool {
		_, ok := g.ids[gamepad.id]
		return !ok
	})

	return nil
}

type nativeGamepad struct {
	id       int
	standard bool

	axisValues    []float64
	buttonPressed []bool
	buttonValues  []float64
}

func (*nativeGamepad) update(gamepad *gamepads) error {
	return nil
}

func (g *nativeGamepad) hasOwnStandardLayoutMapping() bool {
	return g.standard
}

func (g *nativeGamepad) axisCount() int {
	return len(g.axisValues)
}

func (g *nativeGamepad) buttonCount() int {
	return len(g.buttonValues)
}

func (g *nativeGamepad) hatCount() int {
	return 0
}

func (g *nativeGamepad) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axisValues) {
		return 0
	}
	return g.axisValues[axis]
}

func (g *nativeGamepad) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttonPressed) {
		return false
	}
	return g.buttonPressed[button]
}

func (g *nativeGamepad) buttonValue(button int) float64 {
	if button < 0 || button >= len(g.buttonValues) {
		return 0
	}
	return g.buttonValues[button]
}

func (*nativeGamepad) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepad) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	cbackend.VibrateGamepad(g.id, duration, strongMagnitude, weakMagnitude)
}

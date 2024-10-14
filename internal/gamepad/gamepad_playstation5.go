// Copyright 2023 The Ebitengine Authors
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

//go:build playstation5

package gamepad

// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
//
// #include "gamepad_playstation5.h"
import "C"

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type nativeGamepadsImpl struct {
	gamepads []C.struct_Gamepad
	ids      map[int]struct{}
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (*nativeGamepadsImpl) init(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	C.ebitengine_UpdateGamepads()

	g.gamepads = g.gamepads[:0]
	if n := int(C.ebitengine_GetGamepadCount()); n > 0 {
		if cap(g.gamepads) < n {
			g.gamepads = make([]C.struct_Gamepad, n)
		} else {
			g.gamepads = g.gamepads[:n]
		}
		C.ebitengine_GetGamepads(&g.gamepads[0])
	}

	for id := range g.ids {
		delete(g.ids, id)
	}

	for _, gp := range g.gamepads {
		if g.ids == nil {
			g.ids = map[int]struct{}{}
		}
		g.ids[int(gp.id)] = struct{}{}

		gamepad := gamepads.find(func(gamepad *Gamepad) bool {
			return gamepad.native.(*nativeGamepadImpl).id == int(gp.id)
		})
		if gamepad == nil {
			gamepad = gamepads.add("", "")
			gamepad.native = &nativeGamepadImpl{
				id:            int(gp.id),
				axisValues:    make([]float64, gp.axis_count),
				buttonPressed: make([]bool, gp.button_count),
				buttonValues:  make([]float64, gp.button_count),
			}
		}

		gamepad.m.Lock()
		n := gamepad.native.(*nativeGamepadImpl)
		for i := range n.axisValues {
			n.axisValues[i] = float64(gp.axis_values[i])
		}
		for i := range n.buttonValues {
			n.buttonValues[i] = float64(gp.button_values[i])
		}
		for i := range n.buttonPressed {
			n.buttonPressed[i] = gp.button_pressed[i] != 0
		}
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
	id int

	axisValues    []float64
	buttonPressed []bool
	buttonValues  []float64
}

func (*nativeGamepadImpl) update(gamepad *gamepads) error {
	return nil
}

func (g *nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return true
}

func (g *nativeGamepadImpl) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	// TODO: Implement this on the C side.
	if axis < 0 || int(axis) >= len(g.axisValues) {
		return nil
	}
	return axisMappingInput{g: g, axis: int(axis)}
}

func (g *nativeGamepadImpl) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	// TODO: Implement this on the C side.
	if button < 0 || int(button) >= len(g.buttonValues) {
		return nil
	}
	return buttonMappingInput{g: g, button: int(button)}
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

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
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
	C.ebitengine_VibrateGamepad(C.int(g.id), C.double(float64(duration)/float64(time.Second)), C.double(strongMagnitude), C.double(weakMagnitude))
}

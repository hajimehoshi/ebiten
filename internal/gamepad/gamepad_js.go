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
	"encoding/hex"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

var (
	object = js.Global().Get("Object")
)

type nativeGamepadsImpl struct {
	indices map[int]struct{}
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (g *nativeGamepadsImpl) init(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	// TODO: Use the gamepad events instead of navigator.getGamepads.

	defer func() {
		for k := range g.indices {
			delete(g.indices, k)
		}
	}()

	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return nil
	}

	// getGamepads might not exist under a non-secure context (#2100).
	if !nav.Get("getGamepads").Truthy() {
		js.Global().Get("console").Call("warn", "navigator.getGamepads is not available. This might require a secure (HTTPS) context.")
		return nil
	}

	gps := nav.Call("getGamepads")
	if !gps.Truthy() {
		return nil
	}

	l := gps.Length()
	for idx := 0; idx < l; idx++ {
		gp := gps.Index(idx)
		if !gp.Truthy() {
			continue
		}
		index := gp.Get("index").Int()

		if g.indices == nil {
			g.indices = map[int]struct{}{}
		}
		g.indices[index] = struct{}{}

		// The gamepad is not registered yet, register this.
		gamepad := gamepads.find(func(gamepad *Gamepad) bool {
			return index == gamepad.native.(*nativeGamepadImpl).index
		})
		if gamepad == nil {
			name := gp.Get("id").String()

			// This emulates the implementation of EMSCRIPTEN_JoystickGetDeviceGUID.
			// https://github.com/libsdl-org/SDL/blob/0e9560aea22818884921e5e5064953257bfe7fa7/src/joystick/emscripten/SDL_sysjoystick.c#L385
			var sdlID [16]byte
			copy(sdlID[:], []byte(name))

			gamepad = gamepads.add(name, hex.EncodeToString(sdlID[:]))
			gamepad.native = &nativeGamepadImpl{
				index:   index,
				mapping: gp.Get("mapping").String(),
			}
		}
		gamepad.native.(*nativeGamepadImpl).value = gp
	}

	// Remove an unused gamepads.
	gamepads.remove(func(gamepad *Gamepad) bool {
		_, ok := g.indices[gamepad.native.(*nativeGamepadImpl).index]
		return !ok
	})

	return nil
}

type nativeGamepadImpl struct {
	value   js.Value
	index   int
	mapping string
}

func (g *nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return g.mapping == "standard"
}

func (g *nativeGamepadImpl) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	if !g.hasOwnStandardLayoutMapping() {
		return nil
	}
	if axis < 0 || int(axis) >= g.axisCount() {
		return nil
	}
	return axisMappingInput{g: g, axis: int(axis)}
}

func (g *nativeGamepadImpl) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	if !g.hasOwnStandardLayoutMapping() {
		return nil
	}
	if button < 0 || int(button) >= g.buttonCount() {
		return nil
	}
	return buttonMappingInput{g: g, button: int(button)}
}

func (g *nativeGamepadImpl) update(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepadImpl) axisCount() int {
	return g.value.Get("axes").Length()
}

func (g *nativeGamepadImpl) buttonCount() int {
	return g.value.Get("buttons").Length()
}

func (g *nativeGamepadImpl) hatCount() int {
	return 0
}

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	axes := g.value.Get("axes")
	if axis < 0 || axis >= axes.Length() {
		return 0
	}
	return axes.Index(axis).Float()
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	buttons := g.value.Get("buttons")
	if button < 0 || button >= buttons.Length() {
		return 0
	}
	return buttons.Index(button).Get("value").Float()
}

func (g *nativeGamepadImpl) isButtonPressed(button int) bool {
	buttons := g.value.Get("buttons")
	if button < 0 || button >= buttons.Length() {
		return false
	}
	return buttons.Index(button).Get("pressed").Bool()
}

func (g *nativeGamepadImpl) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// vibrationActuator is available on Chrome.
	if va := g.value.Get("vibrationActuator"); va.Truthy() {
		if !va.Get("playEffect").Truthy() {
			return
		}

		prop := object.New()
		prop.Set("startDelay", 0)
		prop.Set("duration", float64(duration/time.Millisecond))
		prop.Set("strongMagnitude", strongMagnitude)
		prop.Set("weakMagnitude", weakMagnitude)
		va.Call("playEffect", "dual-rumble", prop)
		return
	}

	// hapticActuators is available on Firefox.
	if ha := g.value.Get("hapticActuators"); ha.Truthy() {
		// TODO: Is this order correct?
		if ha.Length() > 0 {
			ha.Index(0).Call("pulse", strongMagnitude, float64(duration/time.Millisecond))
		}
		if ha.Length() > 1 {
			ha.Index(1).Call("pulse", weakMagnitude, float64(duration/time.Millisecond))
		}
		return
	}
}

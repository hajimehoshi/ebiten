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
)

var (
	object = js.Global().Get("Object")
	go2cpp = js.Global().Get("go2cpp")
)

type nativeGamepads struct {
	indices map[int]struct{}
}

func (g *nativeGamepads) init(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepads) update(gamepads *gamepads) error {
	// TODO: Use the gamepad events instead of navigator.getGamepads after go2cpp is removed.

	defer func() {
		for k := range g.indices {
			delete(g.indices, k)
		}
	}()

	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
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
			return index == gamepad.index
		})
		if gamepad == nil {
			name := gp.Get("id").String()

			// This emulates the implementation of EMSCRIPTEN_JoystickGetDeviceGUID.
			// https://github.com/libsdl-org/SDL/blob/0e9560aea22818884921e5e5064953257bfe7fa7/src/joystick/emscripten/SDL_sysjoystick.c#L385
			var sdlID [16]byte
			copy(sdlID[:], []byte(name))

			gamepad = gamepads.add(name, hex.EncodeToString(sdlID[:]))
			gamepad.index = index
			gamepad.mapping = gp.Get("mapping").String()
		}
		gamepad.value = gp
	}

	// Remove an unused gamepads.
	gamepads.remove(func(gamepad *Gamepad) bool {
		_, ok := g.indices[gamepad.index]
		return !ok
	})

	return nil
}

type nativeGamepad struct {
	value   js.Value
	index   int
	mapping string
}

func (g *nativeGamepad) hasOwnStandardLayoutMapping() bool {
	// With go2cpp, the controller must have the standard
	if go2cpp.Truthy() {
		return true
	}
	return g.mapping == "standard"
}

func (g *nativeGamepad) update(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepad) axisCount() int {
	return g.value.Get("axes").Length()
}

func (g *nativeGamepad) buttonCount() int {
	return g.value.Get("buttons").Length()
}

func (g *nativeGamepad) hatCount() int {
	return 0
}

func (g *nativeGamepad) axisValue(axis int) float64 {
	axes := g.value.Get("axes")
	if axis < 0 || axis >= axes.Length() {
		return 0
	}
	return axes.Index(axis).Float()
}

func (g *nativeGamepad) buttonValue(button int) float64 {
	buttons := g.value.Get("buttons")
	if button < 0 || button >= buttons.Length() {
		return 0
	}
	return buttons.Index(button).Get("value").Float()
}

func (g *nativeGamepad) isButtonPressed(button int) bool {
	buttons := g.value.Get("buttons")
	if button < 0 || button >= buttons.Length() {
		return false
	}
	return buttons.Index(button).Get("pressed").Bool()
}

func (g *nativeGamepad) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepad) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// vibrationActuator is avaialble on Chrome.
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

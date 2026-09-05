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

// gcControllerToAdd is a controller waiting to be registered, along with the properties read from it
// when it was enumerated or its connect notification arrived.
type gcControllerToAdd struct {
	controller uintptr
	prop       controllerProperty
}

type nativeGamepadsGC struct {
	controllersToAdd    []gcControllerToAdd
	controllersToRemove []uintptr
	controllersMu       sync.Mutex
}

// theGCGamepads is the running GameController backend. The notification blocks reference it
// directly, as theGamepads.native is a composite backend rather than this one.
var theGCGamepads *nativeGamepadsGC

func newNativeGamepadsGC() nativeGamepads {
	return &nativeGamepadsGC{}
}

func (g *nativeGamepadsGC) init(gamepads *gamepads) error {
	theGCGamepads = g

	initializeGCGamepads()
	return nil
}

func (g *nativeGamepadsGC) update(gamepads *gamepads) error {
	g.controllersMu.Lock()
	defer g.controllersMu.Unlock()

	for _, c := range g.controllersToAdd {
		gamepads.addGCGamepad(c.controller, c.prop)
	}
	for _, controller := range g.controllersToRemove {
		gamepads.removeGCGamepad(controller)
	}
	g.controllersToAdd = g.controllersToAdd[:0]
	g.controllersToRemove = g.controllersToRemove[:0]
	return nil
}

type nativeGamepadGC struct {
	controller           uintptr
	buttonMask           uint32
	hasDualshockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool
	leftMotor            *rumbleMotor
	rightMotor           *rumbleMotor
	vibEnd               time.Time

	axes    []float64
	buttons []bool
	hats    []int
}

// close releases g's native resources. close can be called multiple times.
func (g *nativeGamepadGC) close() {
	releaseGCRumbleMotor(g.leftMotor)
	releaseGCRumbleMotor(g.rightMotor)
	g.leftMotor = nil
	g.rightMotor = nil
}

func (g *nativeGamepadGC) update(gamepad *gamepads) error {
	g.updateGCGamepad()
	if !g.vibEnd.IsZero() && time.Since(g.vibEnd) >= 0 {
		vibrateGCGamepad(g.leftMotor, g.rightMotor, 0, 0)
		g.vibEnd = time.Time{}
	}
	return nil
}

func (*nativeGamepadGC) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadGC) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadGC) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadGC) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepadGC) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepadGC) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadGC) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadGC) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadGC) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadGC) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadGC) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return 0
	}
	return g.hats[hat]
}

func (g *nativeGamepadGC) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		g.vibEnd = time.Time{}
		vibrateGCGamepad(g.leftMotor, g.rightMotor, 0, 0)
		return
	}
	g.vibEnd = time.Now().Add(duration)
	vibrateGCGamepad(g.leftMotor, g.rightMotor, strongMagnitude, weakMagnitude)
}

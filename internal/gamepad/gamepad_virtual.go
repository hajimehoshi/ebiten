// Copyright 2026 The Ebitengine Authors
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
	"maps"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

// VirtualGamepadState is the externally supplied state of one virtual gamepad. The raw axes and
// buttons follow the public gamepad view, where hats are folded into buttons; the standard maps hold
// the standard-layout view, with a present key meaning the standard axis or button is available.
type VirtualGamepadState struct {
	ID    ID
	SDLID string
	Name  string

	Axes    []float64
	Buttons []bool

	StandardAxes    map[gamepaddb.StandardAxis]float64
	StandardButtons map[gamepaddb.StandardButton]VirtualStandardGamepadButton
}

// VirtualStandardGamepadButton is one standard-layout button's pressed flag and its analog value in
// 0..1.
type VirtualStandardGamepadButton struct {
	Pressed bool
	Value   float64
}

// VirtualGamepadVibration is a vibration requested by a virtual gamepad: which gamepad (ID), its
// rumble magnitudes in 0..1, and how long they last.
type VirtualGamepadVibration struct {
	ID              ID
	Duration        time.Duration
	StrongMagnitude float64
	WeakMagnitude   float64
}

// AppendVirtualGamepadVibrations appends the vibrations virtual gamepads have requested since the last
// call to dst and returns the extended slice, clearing them so each is reported once. At most one entry
// per gamepad is appended.
func AppendVirtualGamepadVibrations(dst []VirtualGamepadVibration) []VirtualGamepadVibration {
	return theGamepads.appendVirtualVibrations(dst)
}

func (g *gamepads) appendVirtualVibrations(dst []VirtualGamepadVibration) []VirtualGamepadVibration {
	g.m.Lock()
	defer g.m.Unlock()

	for id, gp := range g.gamepads {
		if gp == nil || !gp.virtual {
			continue
		}
		gp.m.Lock()
		n := gp.native.(*nativeGamepadVirtual)
		if n.vibrationPending {
			dst = append(dst, VirtualGamepadVibration{
				ID:              ID(id),
				Duration:        n.vibration.duration,
				StrongMagnitude: n.vibration.strongMagnitude,
				WeakMagnitude:   n.vibration.weakMagnitude,
			})
			n.vibrationPending = false
		}
		gp.m.Unlock()
	}
	return dst
}

// setVirtualGamepads makes the connected gamepads exactly those described by states; a gamepad
// absent from it is disconnected. g.m must be held.
func (g *gamepads) setVirtualGamepads(states []VirtualGamepadState) {
	maxID := -1
	for i := range states {
		if id := int(states[i].ID); id > maxID {
			maxID = id
		}
	}
	for len(g.gamepads) <= maxID {
		g.gamepads = append(g.gamepads, nil)
	}

	// Connect or update every gamepad in the snapshot at its own ID.
	for i := range states {
		s := &states[i]
		gp := g.gamepads[s.ID]
		if gp == nil || !gp.virtual {
			gp = &Gamepad{
				virtual: true,
				native:  &nativeGamepadVirtual{},
			}
			g.gamepads[s.ID] = gp
		}
		gp.setVirtualState(s)
	}

	// Disconnect every gamepad absent from the snapshot.
	for id := range g.gamepads {
		if g.gamepads[id] != nil && !containsVirtualGamepadID(states, ID(id)) {
			g.gamepads[id] = nil
		}
	}
}

func containsVirtualGamepadID(states []VirtualGamepadState, id ID) bool {
	for i := range states {
		if states[i].ID == id {
			return true
		}
	}
	return false
}

func (g *Gamepad) setVirtualState(s *VirtualGamepadState) {
	g.m.Lock()
	defer g.m.Unlock()

	g.name = s.Name
	g.sdlID = s.SDLID
	n := g.native.(*nativeGamepadVirtual)
	// Copy into the gamepad's own (reused) storage so its state does not alias the caller's buffers,
	// which Update is free to reuse or mutate after it returns.
	n.axes = append(n.axes[:0], s.Axes...)
	n.buttons = append(n.buttons[:0], s.Buttons...)
	n.standardAxes = copyMap(n.standardAxes, s.StandardAxes)
	n.standardButtons = copyMap(n.standardButtons, s.StandardButtons)
}

// copyMap returns a copy of src that reuses dst's storage; the result is nil when src is nil.
func copyMap[K comparable, V any](dst, src map[K]V) map[K]V {
	if src == nil {
		return nil
	}
	if dst == nil {
		dst = make(map[K]V, len(src))
	} else {
		clear(dst)
	}
	maps.Copy(dst, src)
	return dst
}

// nativeGamepadsVirtual is a no-op backend: the connected gamepads come from the snapshot
// passed to [Update] (see setVirtualGamepads), not polled from devices.
type nativeGamepadsVirtual struct{}

func (nativeGamepadsVirtual) init(gamepads *gamepads) error {
	return nil
}

func (nativeGamepadsVirtual) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepadVirtual struct {
	axes    []float64
	buttons []bool

	standardAxes    map[gamepaddb.StandardAxis]float64
	standardButtons map[gamepaddb.StandardButton]VirtualStandardGamepadButton

	// vibration holds the latest vibration the game requested; vibrationPending reports that it has not
	// been drained yet. A device has a single current rumble state, so a later request within the same
	// drain interval replaces an earlier one.
	vibration        virtualVibration
	vibrationPending bool
}

// virtualVibration is one requested rumble: its magnitudes and duration.
type virtualVibration struct {
	duration        time.Duration
	strongMagnitude float64
	weakMagnitude   float64
}

func (g *nativeGamepadVirtual) update(gamepads *gamepads) error {
	return nil
}

func (g *nativeGamepadVirtual) hasOwnStandardLayoutMapping() bool {
	return len(g.standardButtons) > 0 || len(g.standardAxes) > 0
}

func (g *nativeGamepadVirtual) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	v, ok := g.standardAxes[axis]
	if !ok {
		return nil
	}
	return virtualStandardAxisMapping{value: v}
}

func (g *nativeGamepadVirtual) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	b, ok := g.standardButtons[button]
	if !ok {
		return nil
	}
	return virtualStandardButtonMapping{pressed: b.Pressed, value: b.Value}
}

func (g *nativeGamepadVirtual) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepadVirtual) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepadVirtual) hatCount() int {
	return 0
}

func (g *nativeGamepadVirtual) isAxisReady(axis int) bool {
	return axis >= 0 && axis < len(g.axes)
}

func (g *nativeGamepadVirtual) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadVirtual) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadVirtual) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadVirtual) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepadVirtual) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// Vibration is guest-to-host feedback: record it so the host can drain and apply it. The gamepad's
	// lock is held by Gamepad.Vibrate, the same lock AppendVirtualGamepadVibrations takes to drain.
	g.vibration = virtualVibration{
		duration:        duration,
		strongMagnitude: strongMagnitude,
		weakMagnitude:   weakMagnitude,
	}
	g.vibrationPending = true
}

// virtualStandardAxisMapping presents a forwarded standard axis value (in -1..1) through the
// mappingInput contract, where StandardAxisValue reads it back as Value()*2-1.
type virtualStandardAxisMapping struct {
	value float64
}

func (m virtualStandardAxisMapping) Pressed() bool {
	return m.value > gamepaddb.ButtonPressedThreshold
}

func (m virtualStandardAxisMapping) Value() float64 {
	return m.value*0.5 + 0.5
}

// virtualStandardButtonMapping presents a forwarded standard button's pressed flag and analog value
// directly.
type virtualStandardButtonMapping struct {
	pressed bool
	value   float64
}

func (m virtualStandardButtonMapping) Pressed() bool {
	return m.pressed
}

func (m virtualStandardButtonMapping) Value() float64 {
	return m.value
}

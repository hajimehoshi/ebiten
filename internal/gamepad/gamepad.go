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
	"errors"
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
	inited       bool
	gamepads     []*Gamepad
	nativeWindow uintptr
	m            sync.Mutex

	native nativeGamepads
}

type nativeGamepads interface {
	init(gamepads *gamepads) error
	update(gamepads *gamepads) error
}

var theGamepads gamepads

// ensureNative constructs the native backend on first use if it is not already set: a virtual backend
// (whose gamepads are passed to [Update]) when virtual is true, otherwise the device-polling backend.
//
// ensureNative must be called with g.m held.
func (g *gamepads) ensureNative(virtual bool) {
	if g.native != nil {
		return
	}
	if virtual {
		g.native = nativeGamepadsVirtual{}
		return
	}
	g.native = newNativeGamepadsImpl()
}

// AppendGamepadIDs is concurrent-safe.
func AppendGamepadIDs(ids []ID) []ID {
	return theGamepads.appendGamepadIDs(ids)
}

// Update is concurrent-safe.
//
// nativeWindow is the platform's native window handle, or 0 if there is none. virtualGamepads selects
// the gamepads: a nil slice polls the real devices, while a non-nil slice (possibly empty) makes the
// connected gamepads exactly those it describes.
func Update(nativeWindow uintptr, virtualGamepads []VirtualGamepadState) error {
	return theGamepads.update(nativeWindow, virtualGamepads)
}

// Get is concurrent-safe.
func Get(id ID) *Gamepad {
	return theGamepads.get(id)
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

func (g *gamepads) update(nativeWindow uintptr, virtualGamepads []VirtualGamepadState) error {
	g.m.Lock()
	defer g.m.Unlock()

	virtual := virtualGamepads != nil

	// The first Update fixes whether the gamepads are virtual; it must not change afterward, as the
	// constructed backend and the gamepad list assume a single mode.
	if g.native != nil {
		if _, wasVirtual := g.native.(nativeGamepadsVirtual); wasVirtual != virtual {
			return errors.New("gamepad: virtualGamepads must be consistently nil or non-nil across Update calls")
		}
	}

	g.ensureNative(virtual)

	if g.nativeWindow != nativeWindow {
		if n, ok := g.native.(interface{ setNativeWindow(uintptr) }); ok {
			n.setNativeWindow(nativeWindow)
		}
		g.nativeWindow = nativeWindow
	}

	if virtual {
		g.setVirtualGamepads(virtualGamepads)
	}

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
	// recognized as gamepads by OSes. In this case, the number of the 'buttons' can exceed the
	// maximum. Skip such devices as a tentative solution (#1173, #2039).
	g.remove(func(gamepad *Gamepad) bool {
		if gamepad.ButtonCount() <= ButtonCount {
			return false
		}
		// The gamepad is never updated again, so release its native resources as a disconnection does.
		gamepad.close()
		return true
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

type Gamepad struct {
	name  string
	sdlID string
	// virtual reports that the gamepad's state is supplied externally (the gamepads passed to [Update])
	// rather than read from a device. Its standard-layout mapping comes from its native input rather
	// than from gamepaddb, so an external driver can present any standard layout it likes regardless of
	// the SDL ID. It is set once when the gamepad is created and never changes.
	virtual bool
	m       sync.Mutex

	native nativeGamepad
}

type mappingInput interface {
	Pressed() bool
	Value() float64 // Normalized to range: 0..1.
}

type axisMappingInput struct {
	g    nativeGamepad
	axis int
}

func (a axisMappingInput) Pressed() bool {
	return a.g.axisValue(a.axis) > gamepaddb.ButtonPressedThreshold
}

func (a axisMappingInput) Value() float64 {
	return a.g.axisValue(a.axis)*0.5 + 0.5
}

type buttonMappingInput struct {
	g      nativeGamepad
	button int
}

func (b buttonMappingInput) Pressed() bool {
	return b.g.isButtonPressed(b.button)
}

func (b buttonMappingInput) Value() float64 {
	return b.g.buttonValue(b.button)
}

type hatMappingInput struct {
	g         nativeGamepad
	hat       int
	direction int
}

func (h hatMappingInput) Pressed() bool {
	return h.g.hatState(h.hat)&h.direction != 0
}

func (h hatMappingInput) Value() float64 {
	if h.Pressed() {
		return 1
	}
	return 0
}

type nativeGamepad interface {
	update(gamepads *gamepads) error
	hasOwnStandardLayoutMapping() bool
	standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput
	standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput
	axisCount() int
	buttonCount() int
	hatCount() int
	isAxisReady(axis int) bool
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

// close releases g's native resources. It does nothing for a backend whose gamepads own none.
func (g *Gamepad) close() {
	g.m.Lock()
	defer g.m.Unlock()

	if n, ok := g.native.(interface{ close() }); ok {
		n.close()
	}
}

// withNative calls f with g's native state while holding g's lock. T must be the concrete native type
// of the backend that owns g.
func withNative[T nativeGamepad](g *Gamepad, f func(n T)) {
	g.m.Lock()
	defer g.m.Unlock()

	f(g.native.(T))
}

// gamepadUnderLock gives gamepaddb a gamepad's raw state without taking the gamepad's lock, so that
// a whole standard-layout query is evaluated in one critical section and reads a single hardware
// report. The gamepad's lock must be held while its methods are called.
type gamepadUnderLock struct {
	gamepad *Gamepad
}

func (g gamepadUnderLock) IsAxisReady(index int) bool {
	return g.gamepad.native.isAxisReady(index)
}

func (g gamepadUnderLock) Axis(index int) float64 {
	return g.gamepad.native.axisValue(index)
}

func (g gamepadUnderLock) Button(index int) bool {
	return g.gamepad.native.isButtonPressed(index)
}

func (g gamepadUnderLock) Hat(index int) int {
	return g.gamepad.native.hatState(index)
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

// IsAxisReady is concurrent-safe.
func (g *Gamepad) IsAxisReady(axis int) bool {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.isAxisReady(axis)
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

// ButtonCountWithHats returns the number of the buttons, counting each hat as the four buttons that
// follow the real ones.
//
// ButtonCountWithHats is concurrent-safe.
func (g *Gamepad) ButtonCountWithHats() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.buttonCount() + g.native.hatCount()*4
}

// IsButtonPressedWithHats reports whether the button is pressed, where button indices at and above
// the number of the real buttons are the hats' directions.
//
// IsButtonPressedWithHats is concurrent-safe.
func (g *Gamepad) IsButtonPressedWithHats(button int) bool {
	g.m.Lock()
	defer g.m.Unlock()

	buttonCount := g.native.buttonCount()
	if button < buttonCount {
		return g.native.isButtonPressed(button)
	}
	if hat := (button - buttonCount) / 4; hat < g.native.hatCount() {
		dir := (button - buttonCount) % 4
		return g.native.hatState(hat)&(1<<dir) != 0
	}
	return false
}

// standardAxisMapping returns the mapping of the standard axis in the gamepad database. The mapping
// is empty for a virtual gamepad, which serves the standard layout its driver gives it.
//
// standardAxisMapping must be called without g's lock, as the database takes its own lock.
func (g *Gamepad) standardAxisMapping(axis gamepaddb.StandardAxis) gamepaddb.Mapping {
	if g.virtual {
		return gamepaddb.Mapping{}
	}
	return gamepaddb.StandardAxisMapping(g.sdlID, axis)
}

// standardButtonMapping returns the mapping of the standard button in the gamepad database. The
// mapping is empty for a virtual gamepad, which serves the standard layout its driver gives it.
//
// standardButtonMapping must be called without g's lock, as the database takes its own lock.
func (g *Gamepad) standardButtonMapping(button gamepaddb.StandardButton) gamepaddb.Mapping {
	if g.virtual {
		return gamepaddb.Mapping{}
	}
	return gamepaddb.StandardButtonMapping(g.sdlID, button)
}

// IsStandardLayoutAvailable is concurrent-safe.
func (g *Gamepad) IsStandardLayoutAvailable() bool {
	if !g.virtual && gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return true
	}

	g.m.Lock()
	defer g.m.Unlock()

	return g.native.hasOwnStandardLayoutMapping()
}

// IsStandardAxisAvailable is concurrent safe.
func (g *Gamepad) IsStandardAxisAvailable(axis gamepaddb.StandardAxis) bool {
	if m := g.standardAxisMapping(axis); m.HasStandardLayout() {
		return m.IsMapped()
	}

	g.m.Lock()
	defer g.m.Unlock()

	return g.native.standardAxisInOwnMapping(axis) != nil
}

// IsStandardButtonAvailable is concurrent safe.
func (g *Gamepad) IsStandardButtonAvailable(button gamepaddb.StandardButton) bool {
	if m := g.standardButtonMapping(button); m.HasStandardLayout() {
		return m.IsMapped()
	}

	g.m.Lock()
	defer g.m.Unlock()

	return g.native.standardButtonInOwnMapping(button) != nil
}

// StandardAxisValue is concurrent-safe.
func (g *Gamepad) StandardAxisValue(axis gamepaddb.StandardAxis) float64 {
	m := g.standardAxisMapping(axis)

	g.m.Lock()
	defer g.m.Unlock()

	if m.HasStandardLayout() {
		return m.AxisValue(gamepadUnderLock{g})
	}
	if mi := g.native.standardAxisInOwnMapping(axis); mi != nil {
		return mi.Value()*2 - 1
	}
	return 0
}

// StandardButtonValue is concurrent-safe.
func (g *Gamepad) StandardButtonValue(button gamepaddb.StandardButton) float64 {
	m := g.standardButtonMapping(button)

	g.m.Lock()
	defer g.m.Unlock()

	if m.HasStandardLayout() {
		return m.ButtonValue(gamepadUnderLock{g})
	}
	if mi := g.native.standardButtonInOwnMapping(button); mi != nil {
		return mi.Value()
	}
	return 0
}

// IsStandardButtonPressed is concurrent-safe.
func (g *Gamepad) IsStandardButtonPressed(button gamepaddb.StandardButton) bool {
	m := g.standardButtonMapping(button)

	g.m.Lock()
	defer g.m.Unlock()

	if m.HasStandardLayout() {
		return m.IsButtonPressed(gamepadUnderLock{g})
	}
	if mi := g.native.standardButtonInOwnMapping(button); mi != nil {
		return mi.Pressed()
	}
	return false
}

// Vibrate is concurrent-safe.
func (g *Gamepad) Vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g.m.Lock()
	defer g.m.Unlock()

	g.native.vibrate(duration, strongMagnitude, weakMagnitude)
}

// motorMagnitude converts a magnitude in the range 0 to 1 to a vibration motor magnitude.
// Out-of-range values are clamped and NaN is treated as 0.
func motorMagnitude(magnitude float64) uint16 {
	// Converting an out-of-range or NaN value to uint16 is implementation-defined,
	// so such values must be rejected before the conversion.
	if !(magnitude > 0) {
		return 0
	}
	if magnitude > 1 {
		return 0xffff
	}
	return uint16(magnitude * 0xffff)
}

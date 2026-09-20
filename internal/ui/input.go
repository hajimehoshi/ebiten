// Copyright 2022 The Ebitengine Authors
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

package ui

import (
	"io/fs"
	"unicode"
)

type MouseButton int

const (
	MouseButton0   MouseButton = iota // The 'left' button
	MouseButton1                      // The 'middle' button
	MouseButton2                      // The 'right' button
	MouseButton3                      // The additional button (usually browser-back)
	MouseButton4                      // The additional button (usually browser-forward)
	MouseButtonMax = MouseButton4
)

// pixelsPerScrollNotch is the estimated scroll amount in device-independent pixels for one notch of a
// typical mouse wheel. It is the amount Chromium scrolls per notch with the Windows default settings.
const pixelsPerScrollNotch = 100

// scrollLinesPerNotch is the number of text lines one notch of a typical mouse wheel scrolls. It is the
// Windows default of SPI_GETWHEELSCROLLLINES.
const scrollLinesPerNotch = 3

// pixelsPerScrollLine is the estimated scroll amount in device-independent pixels for one line of text.
const pixelsPerScrollLine = float64(pixelsPerScrollNotch) / scrollLinesPerNotch

type TouchID int

type Touch struct {
	ID TouchID
	X  float64
	Y  float64
}

// touchIDAllocator maps the IDs a platform assigns to its touches to IDs that are never reused. A
// platform reuses an ID as soon as its touch ends, so consecutive touches can arrive under one
// platform ID; a platform ID that appears in a touch set without having been in the previous one is
// a new touch and gets a fresh ID.
type touchIDAllocator struct {
	// current and previous hold the platform IDs of the current and the previous touch set with the
	// IDs issued for them.
	current  []touchIDMapping
	previous []touchIDMapping

	next TouchID
}

type touchIDMapping struct {
	platformID int
	id         TouchID
}

// nextTouches starts the next set of touches that are down. Each of them must then be passed to id;
// a platform ID that is not passed before the next nextTouches has ended.
func (a *touchIDAllocator) nextTouches() {
	a.current, a.previous = a.previous[:0], a.current
}

// id returns the ID issued for platformID in the current touch set.
func (a *touchIDAllocator) id(platformID int) TouchID {
	if id, ok := lookupTouchIDMapping(a.current, platformID); ok {
		return id
	}
	id, ok := lookupTouchIDMapping(a.previous, platformID)
	if !ok {
		id = a.next
		a.next++
	}
	a.current = append(a.current, touchIDMapping{platformID: platformID, id: id})
	return id
}

func lookupTouchIDMapping(mappings []touchIDMapping, platformID int) (TouchID, bool) {
	for _, m := range mappings {
		if m.platformID == platformID {
			return m.id, true
		}
	}
	return 0, false
}

// touchInClient is a touch whose position is in the platform's client coordinates, pending conversion
// to logical coordinates.
type touchInClient struct {
	id TouchID
	x  float64
	y  float64
}

// LockKeyState is the state of a lock key. The zero value means the platform does not report the state.
type LockKeyState byte

const (
	LockKeyStateUnknown LockKeyState = iota
	LockKeyStateOn
	LockKeyStateOff
)

// NewLockKeyStateFromBool converts an on/off state reported by a platform to a LockKeyState.
func NewLockKeyStateFromBool(on bool) LockKeyState {
	if on {
		return LockKeyStateOn
	}
	return LockKeyStateOff
}

type InputState struct {
	// inputTime belongs to the accumulated state and advances once per consumed snapshot.
	// Recording an event and consuming a snapshot must use the same synchronization.
	inputTime InputTime

	KeyPressedTimes  [KeyMax + 1]InputTime
	KeyReleasedTimes [KeyMax + 1]InputTime

	MouseButtonPressedTimes  [MouseButtonMax + 1]InputTime
	MouseButtonReleasedTimes [MouseButtonMax + 1]InputTime

	CursorX           float64
	CursorY           float64
	WheelX            float64
	WheelY            float64
	ScrollDeltaX      float64
	ScrollDeltaY      float64
	Touches           []Touch
	Runes             []rune
	WindowBeingClosed bool
	DroppedFiles      fs.FS
	CapsLock          LockKeyState
	NumLock           LockKeyState
}

// IsCapsLockOn reports whether Caps Lock is on.
func (i *InputState) IsCapsLockOn() bool {
	if i.CapsLock == LockKeyStateUnknown {
		// An unreported state is off.
		return false
	}
	return i.CapsLock == LockKeyStateOn
}

// IsNumLockOn reports whether the numeric keypad produces digits.
func (i *InputState) IsNumLockOn() bool {
	if i.NumLock == LockKeyStateUnknown {
		// An unreported state is on: a keypad that reports nothing produces digits.
		return true
	}
	return i.NumLock == LockKeyStateOn
}

func (i *InputState) setKeyPressed(key Key, t InputTime) {
	if key < 0 || KeyMax < key {
		return
	}
	i.KeyPressedTimes[key] = t
}

func (i *InputState) setKeyReleased(key Key, t InputTime) {
	if key < 0 || KeyMax < key {
		return
	}
	// Ignore duplicated key releases (#3326).
	if i.KeyPressedTimes[key] <= i.KeyReleasedTimes[key] {
		return
	}
	i.KeyReleasedTimes[key] = t
}

func (i *InputState) setMouseButtonPressed(button MouseButton, t InputTime) {
	if button < 0 || MouseButtonMax < button {
		return
	}
	i.MouseButtonPressedTimes[button] = t
}

func (i *InputState) setMouseButtonReleased(button MouseButton, t InputTime) {
	if button < 0 || MouseButtonMax < button {
		return
	}
	if i.MouseButtonPressedTimes[button] <= i.MouseButtonReleasedTimes[button] {
		return
	}
	i.MouseButtonReleasedTimes[button] = t
}

// releaseAllButtons is called when the browser window loses focus.
func (i *InputState) releaseAllButtons(t InputTime) {
	for j := range i.KeyPressedTimes {
		if i.KeyPressedTimes[Key(j)] <= i.KeyReleasedTimes[Key(j)] {
			continue
		}
		i.KeyReleasedTimes[Key(j)] = t
	}
	for j := range i.MouseButtonPressedTimes {
		if i.MouseButtonPressedTimes[MouseButton(j)] <= i.MouseButtonReleasedTimes[MouseButton(j)] {
			continue
		}
		i.MouseButtonReleasedTimes[j] = t
	}
	i.Touches = i.Touches[:0]
}

func (i *InputState) IsKeyPressed(key Key, tick int64) bool {
	switch key {
	case KeyAlt:
		return i.IsKeyPressed(KeyAltLeft, tick) || i.IsKeyPressed(KeyAltRight, tick)
	case KeyControl:
		return i.IsKeyPressed(KeyControlLeft, tick) || i.IsKeyPressed(KeyControlRight, tick)
	case KeyShift:
		return i.IsKeyPressed(KeyShiftLeft, tick) || i.IsKeyPressed(KeyShiftRight, tick)
	case KeyMeta:
		return i.IsKeyPressed(KeyMetaLeft, tick) || i.IsKeyPressed(KeyMetaRight, tick)
	}

	if key < 0 || KeyMax < key {
		return false
	}
	p := i.KeyPressedTimes[key]
	r := i.KeyReleasedTimes[key]
	if isModifierKey(key) {
		return inputStateModifierPressed(p, r, tick)
	}
	return inputStatePressed(p, r, tick)
}

// A key representing multiple keys like KeyShift is not resolved into its left and right variants
// for the just-pressed, just-released, and duration states: when the variants are pressed or released
// at different ticks, there is no unambiguous tick to report for the combined key.
func (i *InputState) IsKeyJustPressed(key Key, tick int64) bool {
	if key < 0 || KeyMax < key {
		return false
	}
	p := i.KeyPressedTimes[key]
	return inputStateJustPressed(p, tick)
}

func (i *InputState) IsKeyJustReleased(key Key, tick int64) bool {
	if key < 0 || KeyMax < key {
		return false
	}
	r := i.KeyReleasedTimes[key]
	return inputStateJustReleased(r, tick)
}

func (i *InputState) KeyPressDuration(key Key, tick int64) int64 {
	if key < 0 || KeyMax < key {
		return 0
	}
	p := i.KeyPressedTimes[key]
	r := i.KeyReleasedTimes[key]
	if isModifierKey(key) {
		return inputStateModifierDuration(p, r, tick)
	}
	return inputStateDuration(p, r, tick)
}

func (i *InputState) IsMouseButtonPressed(button MouseButton, tick int64) bool {
	if button < 0 || MouseButtonMax < button {
		return false
	}
	p := i.MouseButtonPressedTimes[button]
	r := i.MouseButtonReleasedTimes[button]
	return inputStatePressed(p, r, tick)
}

func (i *InputState) IsMouseButtonJustPressed(button MouseButton, tick int64) bool {
	if button < 0 || MouseButtonMax < button {
		return false
	}
	p := i.MouseButtonPressedTimes[button]
	return inputStateJustPressed(p, tick)
}

func (i *InputState) IsMouseButtonJustReleased(button MouseButton, tick int64) bool {
	if button < 0 || MouseButtonMax < button {
		return false
	}
	r := i.MouseButtonReleasedTimes[button]
	return inputStateJustReleased(r, tick)
}

func (i *InputState) MouseButtonPressDuration(button MouseButton, tick int64) int64 {
	if button < 0 || MouseButtonMax < button {
		return 0
	}
	p := i.MouseButtonPressedTimes[button]
	r := i.MouseButtonReleasedTimes[button]
	return inputStateDuration(p, r, tick)
}

func isModifierKey(key Key) bool {
	switch key {
	case KeyAlt, KeyAltLeft, KeyAltRight,
		KeyControl, KeyControlLeft, KeyControlRight,
		KeyMeta, KeyMetaLeft, KeyMetaRight,
		KeyShift, KeyShiftLeft, KeyShiftRight:
		return true
	}
	return false
}

func inputStatePressed(pressed, released InputTime, tick int64) bool {
	return released < pressed || inputStateJustPressed(pressed, tick)
}

func inputStateJustPressed(pressed InputTime, tick int64) bool {
	return pressed > 0 && pressed.Tick() == tick
}

func inputStateJustReleased(released InputTime, tick int64) bool {
	return released > 0 && released.Tick() == tick
}

// inputStateModifierPressed reports whether a modifier key was down at any point during the tick.
//
// An input event is stamped with the tick it is processed in, so a stalled event queue can deliver a
// modifier's release edge in the same tick as the press edge of the key it qualifies. As a modifier is
// read as a state beside an edge query on that key, ending the press at the release edge would lose the
// chord (#3497, #3498).
func inputStateModifierPressed(pressed, released InputTime, tick int64) bool {
	return inputStatePressed(pressed, released, tick) || inputStateJustReleased(released, tick)
}

func inputStateDuration(pressed, released InputTime, tick int64) int64 {
	if pressed == 0 {
		return 0
	}
	if pressed < released {
		return 0
	}
	return tick - pressed.Tick() + 1
}

func inputStateModifierDuration(pressed, released InputTime, tick int64) int64 {
	if pressed == 0 {
		return 0
	}
	if !inputStateModifierPressed(pressed, released, tick) {
		return 0
	}
	return tick - pressed.Tick() + 1
}

func (i *InputState) nextInputTime() InputTime {
	i.inputTime++
	if i.inputTime.Subtick() == 0 {
		panic("ui: too many input events in a tick")
	}
	return i.inputTime
}

func (i *InputState) copyAndReset(dst *InputState) {
	dst.KeyPressedTimes = i.KeyPressedTimes
	dst.KeyReleasedTimes = i.KeyReleasedTimes
	dst.MouseButtonPressedTimes = i.MouseButtonPressedTimes
	dst.MouseButtonReleasedTimes = i.MouseButtonReleasedTimes
	dst.CursorX = i.CursorX
	dst.CursorY = i.CursorY
	dst.WheelX = i.WheelX
	dst.WheelY = i.WheelY
	dst.ScrollDeltaX = i.ScrollDeltaX
	dst.ScrollDeltaY = i.ScrollDeltaY
	dst.Touches = append(dst.Touches[:0], i.Touches...)
	dst.Runes = append(dst.Runes[:0], i.Runes...)
	dst.WindowBeingClosed = i.WindowBeingClosed
	dst.DroppedFiles = i.DroppedFiles
	dst.CapsLock = i.CapsLock
	dst.NumLock = i.NumLock

	// Reset the members that are updated by deltas, rather than absolute values.
	i.WheelX = 0
	i.WheelY = 0
	i.ScrollDeltaX = 0
	i.ScrollDeltaY = 0
	i.Runes = i.Runes[:0]

	// Reset the members that are never reset until they are explicitly done.
	i.WindowBeingClosed = false
	i.DroppedFiles = nil

	// The next event belongs to the first tick that can consume it.
	i.inputTime = NewInputTimeFromTick(i.inputTime.Tick() + 1)
}

func (i *InputState) appendRune(r rune) {
	if !unicode.IsPrint(r) {
		return
	}
	i.Runes = append(i.Runes, r)
}

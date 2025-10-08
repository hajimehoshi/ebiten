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
	MouseButton1                      // The 'right' button
	MouseButton2                      // The 'middle' button
	MouseButton3                      // The additional button (usually browser-back)
	MouseButton4                      // The additional button (usually browser-forward)
	MouseButtonMax = MouseButton4
)

type TouchID int

type Touch struct {
	ID TouchID
	X  int
	Y  int
}

type InputState struct {
	KeyPressedTimes  [KeyMax + 1]InputTime
	KeyReleasedTimes [KeyMax + 1]InputTime

	MouseButtonPressedTimes  [MouseButtonMax + 1]InputTime
	MouseButtonReleasedTimes [MouseButtonMax + 1]InputTime

	CursorX           float64
	CursorY           float64
	WheelX            float64
	WheelY            float64
	Touches           []Touch
	Runes             []rune
	WindowBeingClosed bool
	DroppedFiles      fs.FS
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
	return inputStatePressed(p, r, tick)
}

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

func inputStatePressed(pressed, released InputTime, tick int64) bool {
	return released < pressed || inputStateJustPressed(pressed, tick)
}

func inputStateJustPressed(pressed InputTime, tick int64) bool {
	return pressed > 0 && pressed.Tick() == tick
}

func inputStateJustReleased(released InputTime, tick int64) bool {
	return released > 0 && released.Tick() == tick
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

func (i *InputState) copyAndReset(dst *InputState) {
	dst.KeyPressedTimes = i.KeyPressedTimes
	dst.KeyReleasedTimes = i.KeyReleasedTimes
	dst.MouseButtonPressedTimes = i.MouseButtonPressedTimes
	dst.MouseButtonReleasedTimes = i.MouseButtonReleasedTimes
	dst.CursorX = i.CursorX
	dst.CursorY = i.CursorY
	dst.WheelX = i.WheelX
	dst.WheelY = i.WheelY
	dst.Touches = append(dst.Touches[:0], i.Touches...)
	dst.Runes = append(dst.Runes[:0], i.Runes...)
	dst.WindowBeingClosed = i.WindowBeingClosed
	dst.DroppedFiles = i.DroppedFiles

	// Reset the members that are updated by deltas, rather than absolute values.
	i.WheelX = 0
	i.WheelY = 0
	i.Runes = i.Runes[:0]

	// Reset the members that are never reset until they are explicitly done.
	i.WindowBeingClosed = false
	i.DroppedFiles = nil
}

func (i *InputState) appendRune(r rune) {
	if !unicode.IsPrint(r) {
		return
	}
	i.Runes = append(i.Runes, r)
}

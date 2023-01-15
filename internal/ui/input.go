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
	Valid bool
	ID    TouchID
	X     int
	Y     int
}

type InputState struct {
	KeyPressed         [KeyMax + 1]bool
	MouseButtonPressed [MouseButtonMax + 1]bool
	CursorX            int
	CursorY            int
	WheelX             float64
	WheelY             float64
	Touches            [16]Touch
	Runes              [16]rune
	RunesCount         int
}

func (i *InputState) resetForTick() {
	i.WheelX = 0
	i.WheelY = 0
	i.RunesCount = 0
}

func (i *InputState) appendRune(r rune) {
	if !unicode.IsPrint(r) {
		return
	}
	if i.RunesCount >= len(i.Runes) {
		return
	}

	i.Runes[i.RunesCount] = r
	i.RunesCount++
}

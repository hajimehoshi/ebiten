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
	KeyPressed         [KeyMax + 1]bool
	MouseButtonPressed [MouseButtonMax + 1]bool
	CursorX            float64
	CursorY            float64
	WheelX             float64
	WheelY             float64
	Touches            []Touch
	Runes              []rune
	WindowBeingClosed  bool
	DroppedFiles       fs.FS
}

func (i *InputState) copyAndReset(dst *InputState) {
	dst.KeyPressed = i.KeyPressed
	dst.MouseButtonPressed = i.MouseButtonPressed
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

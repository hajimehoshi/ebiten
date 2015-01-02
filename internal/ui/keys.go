// Copyright 2015 Hajime Hoshi
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

type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyMax
)

type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	MouseButtonMax
)

type input struct {
	keyPressed         [KeyMax]bool
	mouseButtonPressed [MouseButtonMax]bool
	cursorX            int
	cursorY            int
}

func (i *input) isKeyPressed(key Key) bool {
	return i.keyPressed[key]
}

func (i *input) isMouseButtonPressed(button MouseButton) bool {
	return i.mouseButtonPressed[button]
}

func (i *input) cursorPosition() (x, y int) {
	return i.cursorX, i.cursorY
}

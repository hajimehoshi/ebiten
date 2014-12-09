/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

type Key int

// TODO: Add more keys.
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

type Input interface {
	IsKeyPressed(key Key) bool
	CursorPosition() (x, y int)
	IsMouseButtonPressed(mouseButton MouseButton) bool
}

var currentInput Input

func SetInput(input Input) {
	currentInput = input
}

func IsKeyPressed(key Key) bool {
	if currentInput == nil {
		panic("ebiten.IsKeyPressed: currentInput is not set")
	}
	return currentInput.IsKeyPressed(key)
}

func CursorPosition() (x, y int) {
	if currentInput == nil {
		panic("ebiten.CurrentPosition: currentInput is not set")
	}
	return currentInput.CursorPosition()
}

func IsMouseButtonPressed(button MouseButton) bool {
	if currentInput == nil {
		panic("ebiten.IsMouseButtonPressed: currentInput is not set")
	}
	return currentInput.IsMouseButtonPressed(button)
}

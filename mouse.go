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

type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	MouseButtonMax
)

var currentMouse Mouse

type Mouse interface {
	CursorPosition() (x, y int)
	IsMouseButtonPressed(mouseButton MouseButton) bool
}

func SetMouse(mouse Mouse) {
	currentMouse = mouse
}

func CursorPosition() (x, y int) {
	if currentMouse == nil {
		panic("input.CurrentPosition: currentMouse is not set")
	}
	return currentMouse.CursorPosition()
}

func IsMouseButtonPressed(button MouseButton) bool {
	if currentMouse == nil {
		panic("input.IsMouseButtonPressed: currentMouse is not set")
	}
	return currentMouse.IsMouseButtonPressed(button)
}

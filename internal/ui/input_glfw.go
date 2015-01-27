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

// +build !js

package ui

import (
	glfw "github.com/go-gl/glfw3"
	"math"
)

func updateInput(window *glfw.Window, scale int) error {
	return currentInput.update(window, scale)
}

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]MouseButton{
	glfw.MouseButtonLeft:   MouseButtonLeft,
	glfw.MouseButtonRight:  MouseButtonRight,
	glfw.MouseButtonMiddle: MouseButtonMiddle,
}

func (i *input) update(window *glfw.Window, scale int) error {
	for g, e := range glfwKeyCodeToKey {
		i.keyPressed[e] = window.GetKey(g) == glfw.Press
	}
	for g, e := range glfwMouseButtonToMouseButton {
		i.mouseButtonPressed[e] = window.GetMouseButton(g) == glfw.Press
	}
	x, y := window.GetCursorPosition()
	i.cursorX = int(math.Floor(x)) / scale
	i.cursorY = int(math.Floor(y)) / scale
	for id := glfw.Joystick(0); id < glfw.Joystick(len(i.gamepads)); id++ {
		if !glfw.JoystickPresent(id) {
			continue
		}
		axes32, err := glfw.GetJoystickAxes(id)
		if err != nil {
			return err
		}
		i.gamepads[id].axisNum = len(axes32)
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if len(axes32) <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = float64(axes32[a])
		}
		buttons, err := glfw.GetJoystickButtons(id)
		if err != nil {
			return err
		}
		i.gamepads[id].buttonNum = len(buttons)
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if len(buttons) <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = glfw.Action(buttons[b]) == glfw.Press
		}
	}
	return nil
}

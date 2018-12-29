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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package input

import (
	"sync"
	"unicode"

	"github.com/hajimehoshi/ebiten/internal/glfw"
)

type Input struct {
	keyPressed           map[glfw.Key]bool
	mouseButtonPressed   map[glfw.MouseButton]bool
	callbacksInitialized bool
	scrollX              float64
	scrollY              float64
	cursorX              int
	cursorY              int
	gamepads             [16]gamePad
	touches              []*Touch // This is not updated until GLFW 3.3 is available (#417)
	runeBuffer           []rune
	m                    sync.RWMutex
}

func (i *Input) RuneBuffer() []rune {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.runeBuffer
}

func (i *Input) ClearRuneBuffer() {
	i.m.RLock()
	defer i.m.RUnlock()
	i.runeBuffer = i.runeBuffer[:0]
}

func (i *Input) ResetScrollValues() {
	i.m.RLock()
	defer i.m.RUnlock()
	i.scrollX, i.scrollY = 0, 0
}

func (i *Input) IsKeyPressed(key Key) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if i.keyPressed == nil {
		i.keyPressed = map[glfw.Key]bool{}
	}
	for gk, k := range glfwKeyCodeToKey {
		if k != key {
			continue
		}
		if i.keyPressed[gk] {
			return true
		}
	}
	return false
}

func (i *Input) IsMouseButtonPressed(button MouseButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[glfw.MouseButton]bool{}
	}
	for gb, b := range glfwMouseButtonToMouseButton {
		if b != button {
			continue
		}
		if i.mouseButtonPressed[gb] {
			return true
		}
	}
	return false
}

func (i *Input) Wheel() (xoff, yoff float64) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.scrollX, i.scrollY
}

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]MouseButton{
	glfw.MouseButtonLeft:   MouseButtonLeft,
	glfw.MouseButtonRight:  MouseButtonRight,
	glfw.MouseButtonMiddle: MouseButtonMiddle,
}

func (i *Input) Update(window *glfw.Window, scale float64) {
	i.m.Lock()
	defer i.m.Unlock()
	if !i.callbacksInitialized {
		i.runeBuffer = make([]rune, 0, 1024)
		window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
			if unicode.IsPrint(char) {
				i.m.Lock()
				i.runeBuffer = append(i.runeBuffer, char)
				i.m.Unlock()
			}
		})
		window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
			i.m.Lock()
			i.scrollX = xoff
			i.scrollY = yoff
			i.m.Unlock()
		})
		i.callbacksInitialized = true
	}
	if i.keyPressed == nil {
		i.keyPressed = map[glfw.Key]bool{}
	}
	for gk := range glfwKeyCodeToKey {
		i.keyPressed[gk] = window.GetKey(gk) == glfw.Press
	}
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[glfw.MouseButton]bool{}
	}
	for gb := range glfwMouseButtonToMouseButton {
		i.mouseButtonPressed[gb] = window.GetMouseButton(gb) == glfw.Press
	}
	x, y := window.GetCursorPos()
	i.cursorX = int(x / scale)
	i.cursorY = int(y / scale)
	for id := glfw.Joystick(0); id < glfw.Joystick(len(i.gamepads)); id++ {
		i.gamepads[id].valid = false
		if !glfw.JoystickPresent(id) {
			continue
		}
		i.gamepads[id].valid = true

		axes32 := glfw.GetJoystickAxes(id)
		i.gamepads[id].axisNum = len(axes32)
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if len(axes32) <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = float64(axes32[a])
		}
		buttons := glfw.GetJoystickButtons(id)
		i.gamepads[id].buttonNum = len(buttons)
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if len(buttons) <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = glfw.Action(buttons[b]) == glfw.Press
		}
	}
}

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

//go:build !android && !ios && !js && !nintendosdk

package ui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]MouseButton{
	glfw.MouseButtonLeft:   MouseButton0,
	glfw.MouseButtonMiddle: MouseButton1,
	glfw.MouseButtonRight:  MouseButton2,
	glfw.MouseButton3:      MouseButton3,
	glfw.MouseButton4:      MouseButton4,
}

func (u *userInterfaceImpl) registerInputCallbacks() {
	u.window.SetCharModsCallback(glfw.ToCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()
		u.inputState.appendRune(char)
	}))
	u.window.SetScrollCallback(glfw.ToScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()
		u.inputState.WheelX += xoff
		u.inputState.WheelY += yoff
	}))
}

func (u *userInterfaceImpl) updateInputState() error {
	var err error
	u.mainThread.Call(func() {
		err = u.updateInputStateImpl()
	})
	return err
}

// updateInputStateImpl must be called from the main thread.
func (u *userInterfaceImpl) updateInputStateImpl() error {
	u.m.Lock()
	defer u.m.Unlock()

	for uk, gk := range uiKeyToGLFWKey {
		u.inputState.KeyPressed[uk] = u.window.GetKey(gk) == glfw.Press
	}
	for gb, ub := range glfwMouseButtonToMouseButton {
		u.inputState.MouseButtonPressed[ub] = u.window.GetMouseButton(gb) == glfw.Press
	}

	m := u.currentMonitor()
	s := u.deviceScaleFactor(m)

	cx, cy := u.savedCursorX, u.savedCursorY
	defer func() {
		u.savedCursorX = math.NaN()
		u.savedCursorY = math.NaN()
	}()

	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		cx2, cy2 := u.context.logicalPositionToClientPosition(cx, cy, s)
		cx2 = u.dipToGLFWPixel(cx2, m)
		cy2 = u.dipToGLFWPixel(cy2, m)
		u.window.SetCursorPos(cx2, cy2)
	} else {
		cx2, cy2 := u.window.GetCursorPos()
		cx2 = u.dipFromGLFWPixel(cx2, m)
		cy2 = u.dipFromGLFWPixel(cy2, m)
		cx, cy = u.context.clientPositionToLogicalPosition(cx2, cy2, s)
	}

	// AdjustPosition can return NaN at the initialization.
	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		u.inputState.CursorX, u.inputState.CursorY = cx, cy
	}

	if err := gamepad.Update(); err != nil {
		return err
	}
	return nil
}

func KeyName(key Key) string {
	return theUI.keyName(key)
}

func (u *userInterfaceImpl) keyName(key Key) string {
	if !u.isRunning() {
		return ""
	}

	gk, ok := uiKeyToGLFWKey[key]
	if !ok {
		return ""
	}

	var name string
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		name = glfw.GetKeyName(gk, 0)
	})
	return name
}

func (u *userInterfaceImpl) saveCursorPosition() {
	u.m.Lock()
	defer u.m.Unlock()

	u.savedCursorX = u.inputState.CursorX
	u.savedCursorY = u.inputState.CursorY
}

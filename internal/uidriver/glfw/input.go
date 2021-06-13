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
// +build !android
// +build !ios

package glfw

import (
	"math"
	"sync"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type gamePad struct {
	valid         bool
	guid          string
	name          string
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

type Input struct {
	keyPressed         map[glfw.Key]bool
	mouseButtonPressed map[glfw.MouseButton]bool
	onceCallback       sync.Once
	scrollX            float64
	scrollY            float64
	cursorX            int
	cursorY            int
	gamepads           [16]gamePad
	touches            map[driver.TouchID]pos // TODO: Implement this (#417)
	runeBuffer         []rune
	ui                 *UserInterface
}

type pos struct {
	X int
	Y int
}

func (i *Input) CursorPosition() (x, y int) {
	if !i.ui.isRunning() {
		return 0, 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	return i.cursorX, i.cursorY
}

func (i *Input) GamepadIDs() []driver.GamepadID {
	if !i.ui.isRunning() {
		return nil
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	r := make([]driver.GamepadID, 0, len(i.gamepads))
	for id, g := range i.gamepads {
		if g.valid {
			r = append(r, driver.GamepadID(id))
		}
	}
	return r
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	if !i.ui.isRunning() {
		return ""
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return ""
	}
	return i.gamepads[id].guid
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	if !i.ui.isRunning() {
		return ""
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return ""
	}
	return i.gamepads[id].name
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *Input) GamepadAxis(id driver.GamepadID, axis int) float64 {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	if !i.ui.isRunning() {
		return false
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}

func (i *Input) TouchIDs() []driver.TouchID {
	if !i.ui.isRunning() {
		return nil
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.touches) == 0 {
		return nil
	}
	ids := make([]driver.TouchID, 0, len(i.touches))
	for id := range i.touches {
		ids = append(ids, id)
	}
	return ids
}

func (i *Input) TouchPosition(id driver.TouchID) (x, y int) {
	if !i.ui.isRunning() {
		return 0, 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	for tid, pos := range i.touches {
		if id == tid {
			return pos.X, pos.Y
		}
	}
	return 0, 0
}

func (i *Input) RuneBuffer() []rune {
	if !i.ui.isRunning() {
		return nil
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	rs := make([]rune, len(i.runeBuffer))
	copy(rs, i.runeBuffer)
	return rs
}

func (i *Input) resetForFrame() {
	if !i.ui.isRunning() {
		return
	}

	i.ui.m.Lock()
	defer i.ui.m.Unlock()
	i.runeBuffer = i.runeBuffer[:0]
	i.scrollX, i.scrollY = 0, 0
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	if !i.ui.isRunning() {
		return false
	}

	i.ui.m.Lock()
	defer i.ui.m.Unlock()
	if i.keyPressed == nil {
		i.keyPressed = map[glfw.Key]bool{}
	}
	gk, ok := driverKeyToGLFWKey[key]
	return ok && i.keyPressed[gk]
}

func (i *Input) IsMouseButtonPressed(button driver.MouseButton) bool {
	if !i.ui.isRunning() {
		return false
	}

	i.ui.m.Lock()
	defer i.ui.m.Unlock()
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
	if !i.ui.isRunning() {
		return 0, 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	return i.scrollX, i.scrollY
}

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]driver.MouseButton{
	glfw.MouseButtonLeft:   driver.MouseButtonLeft,
	glfw.MouseButtonRight:  driver.MouseButtonRight,
	glfw.MouseButtonMiddle: driver.MouseButtonMiddle,
}

// update must be called from the main thread.
func (i *Input) update(window *glfw.Window, context driver.UIContext) {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	i.onceCallback.Do(func() {
		window.SetCharModsCallback(glfw.ToCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
			// As this function is called from GLFW callbacks, the current thread is main.
			if !unicode.IsPrint(char) {
				return
			}

			i.ui.m.Lock()
			defer i.ui.m.Unlock()
			i.runeBuffer = append(i.runeBuffer, char)
		}))
		window.SetScrollCallback(glfw.ToScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
			// As this function is called from GLFW callbacks, the current thread is main.
			i.ui.m.Lock()
			defer i.ui.m.Unlock()
			i.scrollX = xoff
			i.scrollY = yoff
		}))
	})
	if i.keyPressed == nil {
		i.keyPressed = map[glfw.Key]bool{}
	}
	for gk := range glfwKeyToDriverKey {
		i.keyPressed[gk] = window.GetKey(gk) == glfw.Press
	}
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[glfw.MouseButton]bool{}
	}
	for gb := range glfwMouseButtonToMouseButton {
		i.mouseButtonPressed[gb] = window.GetMouseButton(gb) == glfw.Press
	}
	cx, cy := window.GetCursorPos()
	// TODO: This is tricky. Rename the function?
	s := i.ui.deviceScaleFactor()
	cx = fromGLFWMonitorPixel(cx, s)
	cy = fromGLFWMonitorPixel(cy, s)
	cx, cy = context.AdjustPosition(cx, cy, s)

	// AdjustPosition can return NaN at the initialization.
	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		i.cursorX, i.cursorY = int(cx), int(cy)
	}

	for id := glfw.Joystick(0); id < glfw.Joystick(len(i.gamepads)); id++ {
		i.gamepads[id].valid = false
		if !id.Present() {
			continue
		}

		buttons := id.GetButtons()

		// A gamepad can be detected even though there are not. Apparently, some special devices are
		// recognized as gamepads by GLFW. In this case, the number of the 'buttons' can exceeds the
		// maximum. Skip such devices as a tentative solution (#1173).
		if len(buttons) > driver.GamepadButtonNum {
			continue
		}

		i.gamepads[id].valid = true

		i.gamepads[id].buttonNum = len(buttons)
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if len(buttons) <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = glfw.Action(buttons[b]) == glfw.Press
		}

		axes32 := id.GetAxes()
		i.gamepads[id].axisNum = len(axes32)
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if len(axes32) <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = float64(axes32[a])
		}

		// Note that GLFW's gamepad GUID follows SDL's GUID.
		i.gamepads[id].guid = id.GetGUID()
		i.gamepads[id].name = id.GetName()
	}
}

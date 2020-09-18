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

package glfw

import (
	"sync"
	"unicode"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/glfw"
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
	touches            map[int]pos // TODO: Implement this (#417)
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
	var cx, cy int
	_ = i.ui.t.Call(func() error {
		cx, cy = i.cursorX, i.cursorY
		return nil
	})
	return cx, cy
}

func (i *Input) GamepadIDs() []int {
	if !i.ui.isRunning() {
		return nil
	}
	var r []int
	_ = i.ui.t.Call(func() error {
		for id, g := range i.gamepads {
			if g.valid {
				r = append(r, id)
			}
		}
		return nil
	})
	return r
}

func (i *Input) GamepadSDLID(id int) string {
	if !i.ui.isRunning() {
		return ""
	}
	var r string
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].guid
		return nil
	})
	return r
}

func (i *Input) GamepadName(id int) string {
	if !i.ui.isRunning() {
		return ""
	}
	var r string
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].name
		return nil
	})
	return r
}

func (i *Input) GamepadAxisNum(id int) int {
	if !i.ui.isRunning() {
		return 0
	}
	var r int
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].axisNum
		return nil
	})
	return r
}

func (i *Input) GamepadAxis(id int, axis int) float64 {
	if !i.ui.isRunning() {
		return 0
	}
	var r float64
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].axes[axis]
		return nil
	})
	return r
}

func (i *Input) GamepadButtonNum(id int) int {
	if !i.ui.isRunning() {
		return 0
	}
	var r int
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].buttonNum
		return nil
	})
	return r
}

func (i *Input) IsGamepadButtonPressed(id int, button driver.GamepadButton) bool {
	if !i.ui.isRunning() {
		return false
	}
	var r bool
	_ = i.ui.t.Call(func() error {
		if len(i.gamepads) <= id {
			return nil
		}
		r = i.gamepads[id].buttonPressed[button]
		return nil
	})
	return r
}

func (i *Input) TouchIDs() []int {
	if !i.ui.isRunning() {
		return nil
	}
	var ids []int
	_ = i.ui.t.Call(func() error {
		if len(i.touches) == 0 {
			return nil
		}
		for id := range i.touches {
			ids = append(ids, id)
		}
		return nil
	})
	return ids
}

func (i *Input) TouchPosition(id int) (x, y int) {
	if !i.ui.isRunning() {
		return 0, 0
	}
	var found bool
	var p pos
	_ = i.ui.t.Call(func() error {
		for tid, pos := range i.touches {
			if id == tid {
				p = pos
				found = true
				break
			}
		}
		return nil
	})
	if !found {
		return 0, 0
	}
	return p.X, p.Y
}

func (i *Input) RuneBuffer() []rune {
	if !i.ui.isRunning() {
		return nil
	}
	var r []rune
	_ = i.ui.t.Call(func() error {
		r = i.runeBuffer
		return nil
	})
	return r
}

func (i *Input) resetForFrame() {
	if !i.ui.isRunning() {
		return
	}
	_ = i.ui.t.Call(func() error {
		i.runeBuffer = i.runeBuffer[:0]
		i.scrollX, i.scrollY = 0, 0
		return nil
	})
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	if !i.ui.isRunning() {
		return false
	}
	var r bool
	_ = i.ui.t.Call(func() error {
		if i.keyPressed == nil {
			i.keyPressed = map[glfw.Key]bool{}
		}
		gk, ok := driverKeyToGLFWKey[key]
		if ok && i.keyPressed[gk] {
			r = true
			return nil
		}
		return nil
	})
	return r
}

func (i *Input) IsMouseButtonPressed(button driver.MouseButton) bool {
	if !i.ui.isRunning() {
		return false
	}
	var r bool
	_ = i.ui.t.Call(func() error {
		if i.mouseButtonPressed == nil {
			i.mouseButtonPressed = map[glfw.MouseButton]bool{}
		}
		for gb, b := range glfwMouseButtonToMouseButton {
			if b != button {
				continue
			}
			if i.mouseButtonPressed[gb] {
				r = true
				return nil
			}
		}
		return nil
	})
	return r
}

func (i *Input) Wheel() (xoff, yoff float64) {
	if !i.ui.isRunning() {
		return 0, 0
	}
	_ = i.ui.t.Call(func() error {
		xoff, yoff = i.scrollX, i.scrollY
		return nil
	})
	return
}

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]driver.MouseButton{
	glfw.MouseButtonLeft:   driver.MouseButtonLeft,
	glfw.MouseButtonRight:  driver.MouseButtonRight,
	glfw.MouseButtonMiddle: driver.MouseButtonMiddle,
}

func (i *Input) appendRuneBuffer(char rune) {
	// As this function is called from GLFW callbacks, the current thread is main.
	if !unicode.IsPrint(char) {
		return
	}
	i.runeBuffer = append(i.runeBuffer, char)
}

func (i *Input) setWheel(xoff, yoff float64) {
	// As this function is called from GLFW callbacks, the current thread is main.
	i.scrollX = xoff
	i.scrollY = yoff
}

func (i *Input) update(window *glfw.Window, context driver.UIContext) {
	var cx, cy float64
	_ = i.ui.t.Call(func() error {
		i.onceCallback.Do(func() {
			window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
				i.appendRuneBuffer(char)
			})
			window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
				i.setWheel(xoff, yoff)
			})
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
		cx, cy = window.GetCursorPos()
		// TODO: This is tricky. Rename the function?
		cx = i.ui.fromGLFWMonitorPixel(cx)
		cy = i.ui.fromGLFWMonitorPixel(cy)
		return nil
	})

	cx, cy = context.AdjustPosition(cx, cy)

	_ = i.ui.t.Call(func() error {
		i.cursorX, i.cursorY = int(cx), int(cy)

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
		return nil
	})
}

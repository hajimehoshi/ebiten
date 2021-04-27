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

package js

import (
	"encoding/hex"
	"syscall/js"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/jsutil"
)

var (
	stringKeydown    = js.ValueOf("keydown")
	stringKeypress   = js.ValueOf("keypress")
	stringKeyup      = js.ValueOf("keyup")
	stringMousedown  = js.ValueOf("mousedown")
	stringMouseup    = js.ValueOf("mouseup")
	stringMousemove  = js.ValueOf("mousemove")
	stringWheel      = js.ValueOf("wheel")
	stringTouchstart = js.ValueOf("touchstart")
	stringTouchend   = js.ValueOf("touchend")
	stringTouchmove  = js.ValueOf("touchmove")
)

var jsKeys []js.Value

func init() {
	for _, k := range driverKeyToJSKey {
		jsKeys = append(jsKeys, k)
	}
}

func jsKeyToID(key js.Value) int {
	// js.Value cannot be used as a map key.
	// As the number of keys is around 100, just a dumb loop should work.
	for i, k := range jsKeys {
		if jsutil.Equal(k, key) {
			return i
		}
	}
	return -1
}

type pos struct {
	X int
	Y int
}

type gamepad struct {
	name          string
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

type Input struct {
	keyPressed         map[int]bool
	keyPressedEdge     map[int]bool
	mouseButtonPressed map[int]bool
	cursorX            int
	cursorY            int
	origCursorX        int
	origCursorY        int
	wheelX             float64
	wheelY             float64
	gamepads           map[driver.GamepadID]gamepad
	touches            map[driver.TouchID]pos
	runeBuffer         []rune
	ui                 *UserInterface
}

func (i *Input) CursorPosition() (x, y int) {
	if i.ui.context == nil {
		return 0, 0
	}
	xf, yf := i.ui.context.AdjustPosition(float64(i.cursorX), float64(i.cursorY), i.ui.DeviceScaleFactor())
	return int(xf), int(yf)
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	// This emulates the implementation of EMSCRIPTEN_JoystickGetDeviceGUID.
	// https://hg.libsdl.org/SDL/file/bc90ce38f1e2/src/joystick/emscripten/SDL_sysjoystick.c#l385
	g, ok := i.gamepads[id]
	if !ok {
		return ""
	}
	var sdlid [16]byte
	copy(sdlid[:], []byte(g.name))
	return hex.EncodeToString(sdlid[:])
}

// GamepadName returns a string containing some information about the controller.
// A PS2 controller returned "810-3-USB Gamepad" on Firefox
// A Xbox 360 controller returned "xinput" on Firefox and "Xbox 360 Controller (XInput STANDARD GAMEPAD)" on Chrome
func (i *Input) GamepadName(id driver.GamepadID) string {
	g, ok := i.gamepads[id]
	if !ok {
		return ""
	}
	return g.name
}

func (i *Input) GamepadIDs() []driver.GamepadID {
	if len(i.gamepads) == 0 {
		return nil
	}

	r := make([]driver.GamepadID, 0, len(i.gamepads))
	for id := range i.gamepads {
		r = append(r, id)
	}
	return r
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	g, ok := i.gamepads[id]
	if !ok {
		return 0
	}
	return g.axisNum
}

func (i *Input) GamepadAxis(id driver.GamepadID, axis int) float64 {
	g, ok := i.gamepads[id]
	if !ok {
		return 0
	}
	if g.axisNum <= axis {
		return 0
	}
	return g.axes[axis]
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	g, ok := i.gamepads[id]
	if !ok {
		return 0
	}
	return g.buttonNum
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	g, ok := i.gamepads[id]
	if !ok {
		return false
	}
	if g.buttonNum <= int(button) {
		return false
	}
	return g.buttonPressed[button]
}

func (i *Input) TouchIDs() []driver.TouchID {
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
	d := i.ui.DeviceScaleFactor()
	for tid, pos := range i.touches {
		if id == tid {
			x, y := i.ui.context.AdjustPosition(float64(pos.X), float64(pos.Y), d)
			return int(x), int(y)
		}
	}
	return 0, 0
}

func (i *Input) RuneBuffer() []rune {
	rs := make([]rune, len(i.runeBuffer))
	copy(rs, i.runeBuffer)
	return rs
}

func (i *Input) resetForFrame() {
	i.runeBuffer = nil
	i.wheelX = 0
	i.wheelY = 0
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	if i.keyPressed != nil {
		if i.keyPressed[jsKeyToID(driverKeyToJSKey[key])] {
			return true
		}
	}
	if i.keyPressedEdge != nil {
		for c, k := range edgeKeyCodeToDriverKey {
			if k != key {
				continue
			}
			if i.keyPressedEdge[c] {
				return true
			}
		}
	}
	return false
}

var codeToMouseButton = map[int]driver.MouseButton{
	0: driver.MouseButtonLeft,
	1: driver.MouseButtonMiddle,
	2: driver.MouseButtonRight,
}

func (i *Input) IsMouseButtonPressed(button driver.MouseButton) bool {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	for c, b := range codeToMouseButton {
		if b != button {
			continue
		}
		if i.mouseButtonPressed[c] {
			return true
		}
	}
	return false
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return i.wheelX, i.wheelY
}

func (i *Input) keyDown(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = true
}

func (i *Input) keyUp(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = false
}

func (i *Input) keyDownEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = true
}

func (i *Input) keyUpEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = false
}

func (i *Input) mouseDown(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = true
}

func (i *Input) mouseUp(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = false
}

func (i *Input) updateGamepads() {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return
	}

	if !nav.Get("getGamepads").Truthy() {
		return
	}

	for k := range i.gamepads {
		delete(i.gamepads, k)
	}

	gamepads := nav.Call("getGamepads")
	l := gamepads.Length()
	for idx := 0; idx < l; idx++ {
		gp := gamepads.Index(idx)
		if !gp.Truthy() {
			continue
		}

		id := driver.GamepadID(gp.Get("index").Int())
		g := gamepad{}
		g.name = gp.Get("id").String()

		axes := gp.Get("axes")
		axesNum := axes.Get("length").Int()
		g.axisNum = axesNum
		for a := 0; a < len(g.axes); a++ {
			if axesNum <= a {
				break
			}
			g.axes[a] = axes.Index(a).Float()
		}

		buttons := gp.Get("buttons")
		buttonsNum := buttons.Get("length").Int()
		g.buttonNum = buttonsNum
		for b := 0; b < len(g.buttonPressed); b++ {
			if buttonsNum <= b {
				break
			}
			g.buttonPressed[b] = buttons.Index(b).Get("pressed").Bool()
		}

		if i.gamepads == nil {
			i.gamepads = map[driver.GamepadID]gamepad{}
		}
		i.gamepads[id] = g
	}
}

func (i *Input) updateFromEvent(e js.Value) {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	switch t := e.Get("type"); {
	case jsutil.Equal(t, stringKeydown):
		c := e.Get("code")
		if c.Type() != js.TypeString {
			code := e.Get("keyCode").Int()
			if edgeKeyCodeToDriverKey[code] == driver.KeyArrowUp ||
				edgeKeyCodeToDriverKey[code] == driver.KeyArrowDown ||
				edgeKeyCodeToDriverKey[code] == driver.KeyArrowLeft ||
				edgeKeyCodeToDriverKey[code] == driver.KeyArrowRight ||
				edgeKeyCodeToDriverKey[code] == driver.KeyBackspace ||
				edgeKeyCodeToDriverKey[code] == driver.KeyTab {
				e.Call("preventDefault")
			}
			i.keyDownEdge(code)
			return
		}
		if jsutil.Equal(c, driverKeyToJSKey[driver.KeyArrowUp]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyArrowDown]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyArrowLeft]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyArrowRight]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyBackspace]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyTab]) {
			e.Call("preventDefault")
		}
		i.keyDown(c)
	case jsutil.Equal(t, stringKeypress):
		if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
			i.runeBuffer = append(i.runeBuffer, r)
		}
	case jsutil.Equal(t, stringKeyup):
		if e.Get("code").Type() != js.TypeString {
			// Assume that UA is Edge.
			code := e.Get("keyCode").Int()
			i.keyUpEdge(code)
			return
		}
		i.keyUp(e.Get("code"))
	case jsutil.Equal(t, stringMousedown):
		button := e.Get("button").Int()
		i.mouseDown(button)
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringMouseup):
		button := e.Get("button").Int()
		i.mouseUp(button)
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringMousemove):
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringWheel):
		// TODO: What if e.deltaMode is not DOM_DELTA_PIXEL?
		i.wheelX = -e.Get("deltaX").Float()
		i.wheelY = -e.Get("deltaY").Float()
	case jsutil.Equal(t, stringTouchstart) || jsutil.Equal(t, stringTouchend) || jsutil.Equal(t, stringTouchmove):
		i.updateTouchesFromEvent(e)
	}
}

func (i *Input) setMouseCursorFromEvent(e js.Value) {
	if i.ui.cursorMode == driver.CursorModeCaptured {
		x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
		i.origCursorX, i.origCursorY = x, y
		dx, dy := e.Get("movementX").Int(), e.Get("movementY").Int()
		i.cursorX += dx
		i.cursorY += dy
		return
	}

	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	i.cursorX, i.cursorY = x, y
	i.origCursorX, i.origCursorY = x, y
}

func (i *Input) recoverCursorPosition() {
	i.cursorX, i.cursorY = i.origCursorX, i.origCursorY
}

func (in *Input) updateTouchesFromEvent(e js.Value) {
	j := e.Get("targetTouches")
	for k := range in.touches {
		delete(in.touches, k)
	}
	for i := 0; i < j.Length(); i++ {
		jj := j.Call("item", i)
		id := driver.TouchID(jj.Get("identifier").Int())
		if in.touches == nil {
			in.touches = map[driver.TouchID]pos{}
		}
		in.touches[id] = pos{
			X: jj.Get("clientX").Int(),
			Y: jj.Get("clientY").Int(),
		}
	}
}

func (i *Input) updateForGo2Cpp() {
	if !go2cpp.Truthy() {
		return
	}

	for k := range i.touches {
		delete(i.touches, k)
	}
	touchCount := go2cpp.Get("touchCount").Int()
	for idx := 0; idx < touchCount; idx++ {
		id := go2cpp.Call("getTouchId", idx)
		x := go2cpp.Call("getTouchX", idx)
		y := go2cpp.Call("getTouchY", idx)
		if i.touches == nil {
			i.touches = map[driver.TouchID]pos{}
		}
		i.touches[driver.TouchID(id.Int())] = pos{
			X: x.Int(),
			Y: y.Int(),
		}
	}

	for k := range i.gamepads {
		delete(i.gamepads, k)
	}
	gamepadCount := go2cpp.Get("gamepadCount").Int()
	for idx := 0; idx < gamepadCount; idx++ {
		g := gamepad{}

		// Avoid buggy devices on GLFW (#1173).
		buttonCount := go2cpp.Call("getGamepadButtonCount", idx).Int()
		if buttonCount > len(g.buttonPressed) {
			continue
		}

		axisCount := go2cpp.Call("getGamepadAxisCount", idx).Int()
		if axisCount > len(g.axes) {
			continue
		}

		id := driver.GamepadID(go2cpp.Call("getGamepadId", idx).Int())

		g.buttonNum = buttonCount
		for j := 0; j < buttonCount; j++ {
			g.buttonPressed[j] = go2cpp.Call("isGamepadButtonPressed", idx, j).Bool()
		}

		g.axisNum = axisCount
		for j := 0; j < axisCount; j++ {
			g.axes[j] = go2cpp.Call("getGamepadAxis", idx, j).Float()
		}

		if i.gamepads == nil {
			i.gamepads = map[driver.GamepadID]gamepad{}
		}
		i.gamepads[id] = g
	}
}

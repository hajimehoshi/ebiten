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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/driver"
)

// InputChars return "printable" runes read from the keyboard at the time update is called.
//
// InputChars represents the environment's locale-dependent translation of keyboard
// input to Unicode characters.
//
// IsKeyPressed is based on a mapping of device (US keyboard) codes to input device keys.
// "Control" and modifier keys should be handled with IsKeyPressed.
//
// InputChars is concurrent-safe.
//
// On Android (ebitenmobile), EbitenView must be focusable to enable to handle keyboard keys.
//
// Keyboards don't work on iOS yet (#1090).
func InputChars() []rune {
	rb := uiDriver().Input().RuneBuffer()
	return append(make([]rune, 0, len(rb)), rb...)
}

// IsKeyPressed returns a boolean indicating whether key is pressed.
//
// If you want to know whether the key started being pressed in the current frame,
// use inpututil.IsKeyJustPressed
//
// Known issue: On Edge browser, some keys don't work well:
//
//   - KeyKPEnter and KeyKPEqual are recognized as KeyEnter and KeyEqual.
//   - KeyPrintScreen is only treated at keyup event.
//
// IsKeyPressed is concurrent-safe.
//
// On Android (ebitenmobile), EbitenView must be focusable to enable to handle keyboard keys.
//
// Keyboards don't work on iOS yet (#1090).
func IsKeyPressed(key Key) bool {
	// There are keys that are invalid values as ebiten.Key (e.g., driver.KeyLeftAlt).
	// Skip such values.
	if !key.isValid() {
		return false
	}

	var keys []driver.Key
	switch key {
	case KeyAlt:
		keys = []driver.Key{driver.KeyLeftAlt, driver.KeyRightAlt}
	case KeyControl:
		keys = []driver.Key{driver.KeyLeftControl, driver.KeyRightControl}
	case KeyShift:
		keys = []driver.Key{driver.KeyLeftShift, driver.KeyRightShift}
	default:
		keys = []driver.Key{driver.Key(key)}
	}
	for _, k := range keys {
		if uiDriver().Input().IsKeyPressed(k) {
			return true
		}
	}
	return false
}

// CursorPosition returns a position of a mouse cursor relative to the game screen (window). The cursor position is
// 'logical' position and this considers the scale of the screen.
//
// CursorPosition is concurrent-safe.
func CursorPosition() (x, y int) {
	return uiDriver().Input().CursorPosition()
}

// Wheel returns the x and y offset of the mouse wheel or touchpad scroll.
// It returns 0 if the wheel isn't being rolled.
//
// Wheel is concurrent-safe.
func Wheel() (xoff, yoff float64) {
	return uiDriver().Input().Wheel()
}

// IsMouseButtonPressed returns a boolean indicating whether mouseButton is pressed.
//
// If you want to know whether the mouseButton started being pressed in the current frame,
// use inpututil.IsMouseButtonJustPressed
//
// IsMouseButtonPressed is concurrent-safe.
//
// Note that touch events not longer affect IsMouseButtonPressed's result as of 1.4.0-alpha.
// Use Touches instead.
func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return uiDriver().Input().IsMouseButtonPressed(driver.MouseButton(mouseButton))
}

// GamepadSDLID returns a string with the GUID generated in the same way as SDL.
// To detect devices, see also the community project of gamepad devices database: https://github.com/gabomdq/SDL_GameControllerDB
//
// GamepadSDLID always returns an empty string on browsers and mobiles.
//
// GamepadSDLID is concurrent-safe.
func GamepadSDLID(id int) string {
	return uiDriver().Input().GamepadSDLID(id)
}

// GamepadName returns a string with the name.
// This function may vary in how it returns descriptions for the same device across platforms
// for example the following drivers/platforms see a Xbox One controller as the following:
//
//   - Windows: "Xbox Controller"
//   - Chrome: "Xbox 360 Controller (XInput STANDARD GAMEPAD)"
//   - Firefox: "xinput"
//
// GamepadName always returns an empty string on iOS.
//
// GamepadName is concurrent-safe.
func GamepadName(id int) string {
	return uiDriver().Input().GamepadName(id)
}

// GamepadIDs returns a slice indicating available gamepad IDs.
//
// GamepadIDs is concurrent-safe.
//
// GamepadIDs always returns an empty slice on iOS.
func GamepadIDs() []int {
	return uiDriver().Input().GamepadIDs()
}

// GamepadAxisNum returns the number of axes of the gamepad (id).
//
// GamepadAxisNum is concurrent-safe.
//
// GamepadAxisNum always returns 0 on iOS.
func GamepadAxisNum(id int) int {
	return uiDriver().Input().GamepadAxisNum(id)
}

// GamepadAxis returns the float value [-1.0 - 1.0] of the given gamepad (id)'s axis (axis).
//
// GamepadAxis is concurrent-safe.
//
// GamepadAxis always returns 0 on iOS.
func GamepadAxis(id int, axis int) float64 {
	return uiDriver().Input().GamepadAxis(id, axis)
}

// GamepadButtonNum returns the number of the buttons of the given gamepad (id).
//
// GamepadButtonNum is concurrent-safe.
//
// GamepadButtonNum always returns 0 on iOS.
func GamepadButtonNum(id int) int {
	return uiDriver().Input().GamepadButtonNum(id)
}

// IsGamepadButtonPressed returns the boolean indicating the given button of the gamepad (id) is pressed or not.
//
// If you want to know whether the given button of gamepad (id) started being pressed in the current frame,
// use inpututil.IsGamepadButtonJustPressed
//
// IsGamepadButtonPressed is concurrent-safe.
//
// The relationships between physical buttons and buttion IDs depend on environments.
// There can be differences even between Chrome and Firefox.
//
// IsGamepadButtonPressed always returns false on iOS.
func IsGamepadButtonPressed(id int, button GamepadButton) bool {
	return uiDriver().Input().IsGamepadButtonPressed(id, driver.GamepadButton(button))
}

// TouchIDs returns the current touch states.
//
// If you want to know whether a touch started being pressed in the current frame,
// use inpututil.JustPressedTouchIDs
//
// TouchIDs returns nil when there are no touches.
// TouchIDs always returns nil on desktops.
//
// TouchIDs is concurrent-safe.
func TouchIDs() []int {
	return uiDriver().Input().TouchIDs()
}

// TouchPosition returns the position for the touch of the specified ID.
//
// If the touch of the specified ID is not present, TouchPosition returns (0, 0).
//
// TouchPosition is cuncurrent-safe.
func TouchPosition(id int) (int, int) {
	found := false
	for _, i := range uiDriver().Input().TouchIDs() {
		if id == i {
			found = true
			break
		}
	}
	if !found {
		return 0, 0
	}

	return uiDriver().Input().TouchPosition(id)
}

// Touch represents a touch.
//
// Deprecated: (as of 1.7.0). Use TouchPosition instead.
type Touch interface {
	// ID returns an identifier for one stroke.
	ID() int

	// Position returns the position of the touch.
	Position() (x, y int)
}

type touch struct {
	id int
	x  int
	y  int
}

func (t *touch) ID() int {
	return t.id
}

func (t *touch) Position() (x, y int) {
	return t.x, t.y
}

// Touches returns the current touches.
//
// Deprecated: (as of 1.7.0) Use TouchIDs instead.
func Touches() []Touch {
	var ts []Touch
	for _, id := range TouchIDs() {
		x, y := TouchPosition(id)
		ts = append(ts, &touch{
			id: id,
			x:  x,
			y:  y,
		})
	}
	return ts
}

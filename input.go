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
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

// AppendInputChars appends "printable" runes, read from the keyboard at the time update is called, to runes,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendInputChars represents the environment's locale-dependent translation of keyboard
// input to Unicode characters.
//
// IsKeyPressed is based on a mapping of device (US keyboard) codes to input device keys.
// "Control" and modifier keys should be handled with IsKeyPressed.
//
// AppendInputChars is concurrent-safe.
//
// On Android (ebitenmobile), EbitenView must be focusable to enable to handle keyboard keys.
//
// Keyboards don't work on iOS yet (#1090).
func AppendInputChars(runes []rune) []rune {
	return uiDriver().Input().AppendInputChars(runes)
}

// InputChars return "printable" runes read from the keyboard at the time update is called.
//
// Deprecated: as of v2.2. Use AppendInputChars instead.
func InputChars() []rune {
	return AppendInputChars(nil)
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
	if !key.isValid() {
		return false
	}

	var keys []driver.Key
	switch key {
	case KeyAlt:
		keys = []driver.Key{driver.KeyAltLeft, driver.KeyAltRight}
	case KeyControl:
		keys = []driver.Key{driver.KeyControlLeft, driver.KeyControlRight}
	case KeyShift:
		keys = []driver.Key{driver.KeyShiftLeft, driver.KeyShiftRight}
	case KeyMeta:
		keys = []driver.Key{driver.KeyMetaLeft, driver.KeyMetaRight}
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
// CursorPosition returns (0, 0) before the main loop on desktops and browsers.
//
// CursorPosition always returns (0, 0) on mobiles.
//
// CursorPosition is concurrent-safe.
func CursorPosition() (x, y int) {
	return uiDriver().Input().CursorPosition()
}

// Wheel returns x and y offsets of the mouse wheel or touchpad scroll.
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
func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return uiDriver().Input().IsMouseButtonPressed(mouseButton)
}

// GamepadID represents a gamepad's identifier.
type GamepadID = driver.GamepadID

// GamepadSDLID returns a string with the GUID generated in the same way as SDL.
// To detect devices, see also the community project of gamepad devices database: https://github.com/gabomdq/SDL_GameControllerDB
//
// GamepadSDLID always returns an empty string on browsers and mobiles.
//
// GamepadSDLID is concurrent-safe.
func GamepadSDLID(id GamepadID) string {
	return uiDriver().Input().GamepadSDLID(id)
}

// GamepadName returns a string with the name.
// This function may vary in how it returns descriptions for the same device across platforms.
// for example the following drivers/platforms see a Xbox One controller as the following:
//
//   - Windows: "Xbox Controller"
//   - Chrome: "Xbox 360 Controller (XInput STANDARD GAMEPAD)"
//   - Firefox: "xinput"
//
// GamepadName always returns an empty string on iOS.
//
// GamepadName is concurrent-safe.
func GamepadName(id GamepadID) string {
	return uiDriver().Input().GamepadName(id)
}

// AppendGamepadIDs appends available gamepad IDs to gamepadIDs, and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendGamepadIDs is concurrent-safe.
//
// AppendGamepadIDs doesn't append anything on iOS.
func AppendGamepadIDs(gamepadIDs []GamepadID) []GamepadID {
	return uiDriver().Input().AppendGamepadIDs(gamepadIDs)
}

// GamepadIDs returns a slice indicating available gamepad IDs.
//
// Deprecated: as of v2.2. Use AppendGamepadIDs instead.
func GamepadIDs() []GamepadID {
	return AppendGamepadIDs(nil)
}

// GamepadAxisNum returns the number of axes of the gamepad (id).
//
// GamepadAxisNum is concurrent-safe.
//
// GamepadAxisNum always returns 0 on iOS.
func GamepadAxisNum(id GamepadID) int {
	return uiDriver().Input().GamepadAxisNum(id)
}

// GamepadAxisValue returns a float value [-1.0 - 1.0] of the given gamepad (id)'s axis (axis).
//
// GamepadAxisValue is concurrent-safe.
//
// GamepadAxisValue always returns 0 on iOS.
func GamepadAxisValue(id GamepadID, axis int) float64 {
	return uiDriver().Input().GamepadAxisValue(id, axis)
}

// GamepadAxis returns a float value [-1.0 - 1.0] of the given gamepad (id)'s axis (axis).
//
// Deprecated: as of v2.2. Use GamepadAxisValue instead.
func GamepadAxis(id GamepadID, axis int) float64 {
	return GamepadAxisValue(id, axis)
}

// GamepadButtonNum returns the number of the buttons of the given gamepad (id).
//
// GamepadButtonNum is concurrent-safe.
//
// GamepadButtonNum always returns 0 on iOS.
func GamepadButtonNum(id GamepadID) int {
	return uiDriver().Input().GamepadButtonNum(id)
}

// IsGamepadButtonPressed reports whether the given button of the gamepad (id) is pressed or not.
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
func IsGamepadButtonPressed(id GamepadID, button GamepadButton) bool {
	return uiDriver().Input().IsGamepadButtonPressed(id, button)
}

// StandardGamepadAxisValue returns a float value [-1.0 - 1.0] of the given gamepad (id)'s standard axis (axis).
//
// StandardGamepadAxisValue returns 0 when the gamepad doesn't have a standard gamepad layout mapping.
//
// StandardGamepadAxisValue is concurrent safe.
func StandardGamepadAxisValue(id GamepadID, axis StandardGamepadAxis) float64 {
	return uiDriver().Input().StandardGamepadAxisValue(id, axis)
}

// StandardGamepadButtonValue returns a float value [0.0 - 1.0] of the given gamepad (id)'s standard button (button).
//
// StandardGamepadButtonValue returns 0 when the gamepad doesn't have a standard gamepad layout mapping.
//
// StandardGamepadButtonValue is concurrent safe.
func StandardGamepadButtonValue(id GamepadID, button StandardGamepadButton) float64 {
	return uiDriver().Input().StandardGamepadButtonValue(id, button)
}

// IsStandardGamepadButtonPressed reports whether the given gamepad (id)'s standard gamepad button (button) is pressed.
//
// IsStandardGamepadButtonPressed returns false when the gamepad doesn't have a standard gamepad layout mapping.
//
// IsStandardGamepadButtonPressed is concurrent safe.
func IsStandardGamepadButtonPressed(id GamepadID, button StandardGamepadButton) bool {
	return uiDriver().Input().IsStandardGamepadButtonPressed(id, button)
}

// IsStandardGamepadLayoutAvailable reports whether the gamepad (id) has a standard gamepad layout mapping.
//
// IsStandardGamepadLayoutAvailable is concurrent-safe.
func IsStandardGamepadLayoutAvailable(id GamepadID) bool {
	return uiDriver().Input().IsStandardGamepadLayoutAvailable(id)
}

// UpdateStandardGamepadLayoutMappings parses the specified string mappings in SDL_GameControllerDB format and
// updates the gamepad layout definitions.
//
// UpdateStandardGamepadLayoutMappings reports whether the mappings were applied,
// and returns an error in case any occurred while parsing the mappings.
//
// One or more input definitions can be provided separated by newlines.
// In particular, it is valid to pass an entire gamecontrollerdb.txt file.
// Note though that Ebiten already includes its own copy of this file,
// so this call should only be necessary to add mappings for hardware not supported yet;
// ideally games using the StandardGamepad* functions should allow the user to provide mappings and
// then call this function if provided.
// When using this facility to support new hardware, please also send a pull request to
// https://github.com/gabomdq/SDL_GameControllerDB to make your mapping available to everyone else.
//
// On platforms where gamepad mappings are not managed by Ebiten, this always returns false and nil.
//
// UpdateStandardGamepadLayoutMappings is concurrent-safe.
//
// Updated mappings take effect immediately even for already connected gamepads.
func UpdateStandardGamepadLayoutMappings(mappings string) (bool, error) {
	return gamepaddb.Update([]byte(mappings))
}

// TouchID represents a touch's identifier.
type TouchID = driver.TouchID

// AppendTouchIDs appends the current touch states to touches, and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// If you want to know whether a touch started being pressed in the current frame,
// use inpututil.JustPressedTouchIDs
//
// AppendTouchIDs doesn't append anything when there are no touches.
// AppendTouchIDs always does nothing on desktops.
//
// AppendTouchIDs is concurrent-safe.
func AppendTouchIDs(touches []TouchID) []TouchID {
	return uiDriver().Input().AppendTouchIDs(touches)
}

// TouchIDs returns the current touch states.
//
// Deperecated: as of v2.2. Use AppendTouchIDs instead.
func TouchIDs() []TouchID {
	return AppendTouchIDs(nil)
}

// TouchPosition returns the position for the touch of the specified ID.
//
// If the touch of the specified ID is not present, TouchPosition returns (0, 0).
//
// TouchPosition is cuncurrent-safe.
func TouchPosition(id TouchID) (int, int) {
	return uiDriver().Input().TouchPosition(id)
}

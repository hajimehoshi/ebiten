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
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/inputstate"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// AppendInputChars appends "printable" runes, read from the keyboard at the time Update is called, to runes,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendInputChars represents the environment's locale-dependent translation of keyboard
// input to Unicode characters. On the other hand, Key represents a physical key of US keyboard layout
//
// "Control" and modifier keys should be handled with IsKeyPressed.
//
// AppendInputChars is concurrent-safe.
//
// On Android (ebitenmobile), EbitenView must be focusable to enable to handle keyboard keys.
func AppendInputChars(runes []rune) []rune {
	return inputstate.Get().AppendInputChars(runes)
}

// InputChars return "printable" runes read from the keyboard at the time Update is called.
//
// Deprecated: as of v2.2. Use AppendInputChars instead.
func InputChars() []rune {
	return AppendInputChars(nil)
}

// IsKeyPressed returns a boolean indicating whether key is pressed.
//
// If you want to know whether the key started being pressed in the current tick,
// use inpututil.IsKeyJustPressed
//
// Note that a Key represents a physical key of US keyboard layout.
// For example, KeyQ represents Q key on US keyboards and ' (quote) key on Dvorak keyboards.
//
// IsKeyPressed is concurrent-safe.
//
// On Android (ebitenmobile), EbitenView must be focusable to enable to handle keyboard keys.
func IsKeyPressed(key Key) bool {
	return inputstate.Get().IsKeyPressed(ui.Key(key))
}

// KeyName returns a key name for the current keyboard layout.
// For example, KeyName(KeyQ) returns 'q' for a QWERTY keyboard, and returns 'a' for an AZERTY keyboard.
//
// KeyName returns an empty string if 1) the key doesn't have a physical key name, 2) the platform doesn't support KeyName,
// or 3) the main loop doesn't start yet.
//
// KeyName is supported by desktops and browsers.
//
// KeyName is concurrent-safe.
func KeyName(key Key) string {
	return ui.Get().KeyName(ui.Key(key))
}

// CursorPosition returns a position of a mouse cursor relative to the game screen (window). The cursor position is
// 'logical' position and this considers the scale of the screen.
//
// CursorPosition returns (0, 0) before the main loop on desktops and browsers.
//
// CursorPosition always returns (0, 0) on mobile native applications.
//
// CursorPosition is concurrent-safe.
func CursorPosition() (x, y int) {
	cx, cy := inputstate.Get().CursorPosition()
	return int(cx), int(cy)
}

// Wheel returns x and y offsets of the mouse wheel or touchpad scroll.
// It returns 0 if the wheel isn't being rolled.
//
// Wheel is concurrent-safe.
func Wheel() (xoff, yoff float64) {
	return inputstate.Get().Wheel()
}

// IsMouseButtonPressed returns a boolean indicating whether mouseButton is pressed.
//
// If you want to know whether the mouseButton started being pressed in the current tick,
// use inpututil.IsMouseButtonJustPressed
//
// IsMouseButtonPressed is concurrent-safe.
func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return inputstate.Get().IsMouseButtonPressed(ui.MouseButton(mouseButton))
}

// GamepadID represents a gamepad identifier.
type GamepadID = gamepad.ID

// GamepadSDLID returns a string with the GUID generated in the same way as SDL.
// To detect devices, see also the community project of gamepad devices database: https://github.com/gabomdq/SDL_GameControllerDB
//
// GamepadSDLID always returns an empty string on browsers and mobiles.
//
// GamepadSDLID is concurrent-safe.
func GamepadSDLID(id GamepadID) string {
	g := gamepad.Get(id)
	if g == nil {
		return ""
	}
	return g.SDLID()
}

// GamepadName returns a string with the name.
// This function may vary in how it returns descriptions for the same device across platforms.
// for example the following drivers/platforms see an Xbox One controller as the following:
//
//   - Windows: "Xbox Controller"
//   - Chrome: "Xbox 360 Controller (XInput STANDARD GAMEPAD)"
//   - Firefox: "xinput"
//
// GamepadName is concurrent-safe.
func GamepadName(id GamepadID) string {
	g := gamepad.Get(id)
	if g == nil {
		return ""
	}
	return g.Name()
}

// AppendGamepadIDs appends available gamepad IDs to gamepadIDs, and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendGamepadIDs is concurrent-safe.
func AppendGamepadIDs(gamepadIDs []GamepadID) []GamepadID {
	return gamepad.AppendGamepadIDs(gamepadIDs)
}

// GamepadIDs returns a slice indicating available gamepad IDs.
//
// Deprecated: as of v2.2. Use AppendGamepadIDs instead.
func GamepadIDs() []GamepadID {
	return AppendGamepadIDs(nil)
}

// GamepadAxisCount returns the number of axes of the gamepad (id).
//
// GamepadAxisCount is concurrent-safe.
func GamepadAxisCount(id GamepadID) int {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.AxisCount()
}

// GamepadAxisNum returns the number of axes of the gamepad (id).
//
// Deprecated: as of v2.4. Use GamepadAxisCount instead.
func GamepadAxisNum(id GamepadID) int {
	return GamepadAxisCount(id)
}

// GamepadAxisValue returns a float value [-1.0 - 1.0] of the given gamepad (id)'s axis (axis).
//
// GamepadAxisValue is concurrent-safe.
func GamepadAxisValue(id GamepadID, axis GamepadAxisType) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.Axis(int(axis))
}

// GamepadAxis returns a float value [-1.0 - 1.0] of the given gamepad (id)'s axis (axis).
//
// Deprecated: as of v2.2. Use GamepadAxisValue instead.
func GamepadAxis(id GamepadID, axis GamepadAxisType) float64 {
	return GamepadAxisValue(id, axis)
}

// GamepadButtonCount returns the number of the buttons of the given gamepad (id).
//
// GamepadButtonCount is concurrent-safe.
func GamepadButtonCount(id GamepadID) int {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}

	// For backward compatibility, hats are treated as buttons in GLFW.
	return g.ButtonCount() + g.HatCount()*4
}

// GamepadButtonNum returns the number of the buttons of the given gamepad (id).
//
// Deprecated: as of v2.4. Use GamepadButtonCount instead.
func GamepadButtonNum(id GamepadID) int {
	return GamepadButtonCount(id)
}

// IsGamepadButtonPressed reports whether the given button of the gamepad (id) is pressed or not.
//
// If you want to know whether the given button of gamepad (id) started being pressed in the current tick,
// use inpututil.IsGamepadButtonJustPressed
//
// IsGamepadButtonPressed is concurrent-safe.
//
// The relationships between physical buttons and button IDs depend on environments.
// There can be differences even between Chrome and Firefox.
func IsGamepadButtonPressed(id GamepadID, button GamepadButton) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}

	nbuttons := g.ButtonCount()
	if int(button) < nbuttons {
		return g.Button(int(button))
	}

	// For backward compatibility, hats are treated as buttons in GLFW.
	if hat := (int(button) - nbuttons) / 4; hat < g.HatCount() {
		dir := (int(button) - nbuttons) % 4
		return g.Hat(hat)&(1<<dir) != 0
	}

	return false
}

// StandardGamepadAxisValue returns a float value [-1.0 - 1.0] of the given gamepad (id)'s standard axis (axis).
//
// StandardGamepadAxisValue returns 0 when the gamepad doesn't have a standard gamepad layout mapping.
//
// StandardGamepadAxisValue is concurrent safe.
func StandardGamepadAxisValue(id GamepadID, axis StandardGamepadAxis) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.StandardAxisValue(axis)
}

// StandardGamepadButtonValue returns a float value [0.0 - 1.0] of the given gamepad (id)'s standard button (button).
//
// StandardGamepadButtonValue returns 0 when the gamepad doesn't have a standard gamepad layout mapping.
//
// StandardGamepadButtonValue is concurrent safe.
func StandardGamepadButtonValue(id GamepadID, button StandardGamepadButton) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.StandardButtonValue(button)
}

// IsStandardGamepadButtonPressed reports whether the given gamepad (id)'s standard gamepad button (button) is pressed.
//
// IsStandardGamepadButtonPressed returns false when the gamepad doesn't have a standard gamepad layout mapping.
//
// IsStandardGamepadButtonPressed is concurrent safe.
func IsStandardGamepadButtonPressed(id GamepadID, button StandardGamepadButton) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardButtonPressed(button)
}

// IsStandardGamepadLayoutAvailable reports whether the gamepad (id) has a standard gamepad layout mapping.
//
// IsStandardGamepadLayoutAvailable is concurrent-safe.
func IsStandardGamepadLayoutAvailable(id GamepadID) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardLayoutAvailable()
}

// IsStandardGamepadAxisAvailable reports whether the standard gamepad axis is available on the gamepad (id).
//
// IsStandardGamepadAxisAvailable is concurrent-safe.
func IsStandardGamepadAxisAvailable(id GamepadID, axis StandardGamepadAxis) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardAxisAvailable(axis)
}

// IsStandardGamepadButtonAvailable reports whether the standard gamepad button is available on the gamepad (id).
//
// IsStandardGamepadButtonAvailable is concurrent-safe.
func IsStandardGamepadButtonAvailable(id GamepadID, button StandardGamepadButton) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardButtonAvailable(button)
}

// UpdateStandardGamepadLayoutMappings parses the specified string mappings in SDL_GameControllerDB format and
// updates the gamepad layout definitions.
//
// UpdateStandardGamepadLayoutMappings reports whether the mappings were applied,
// and returns an error in case any occurred while parsing the mappings.
//
// One or more input definitions can be provided separated by newlines.
// In particular, it is valid to pass an entire gamecontrollerdb.txt file.
// Note though that Ebitengine already includes its own copy of this file,
// so this call should only be necessary to add mappings for hardware not supported yet;
// ideally games using the StandardGamepad* functions should allow the user to provide mappings and
// then call this function if provided.
// When using this facility to support new hardware, please also send a pull request to
// https://github.com/gabomdq/SDL_GameControllerDB to make your mapping available to everyone else.
//
// A platform field in a line corresponds with a GOOS like the following:
//
//	"Windows":  GOOS=windows
//	"Mac OS X": GOOS=darwin (not ios)
//	"Linux":    GOOS=linux (not android)
//	"Android":  GOOS=android
//	"iOS":      GOOS=ios
//	"":         Any GOOS
//
// On platforms where gamepad mappings are not managed by Ebitengine, this always returns false and nil.
//
// UpdateStandardGamepadLayoutMappings is concurrent-safe.
//
// UpdateStandardGamepadLayoutMappings mappings take effect immediately even for already connected gamepads.
//
// UpdateStandardGamepadLayoutMappings works atomically. If an error happens, nothing is updated.
func UpdateStandardGamepadLayoutMappings(mappings string) (bool, error) {
	if err := gamepaddb.Update([]byte(mappings)); err != nil {
		return false, err
	}
	return true, nil
}

// TouchID represents a touch's identifier.
type TouchID int

// AppendTouchIDs appends the current touch states to touches, and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// If you want to know whether a touch started being pressed in the current tick,
// use inpututil.JustPressedTouchIDs
//
// AppendTouchIDs doesn't append anything when there are no touches.
// AppendTouchIDs always does nothing on desktops.
//
// AppendTouchIDs is concurrent-safe.
func AppendTouchIDs(touches []TouchID) []TouchID {
	return inputstate.AppendTouchIDs(touches)
}

// TouchIDs returns the current touch states.
//
// Deprecated: as of v2.2. Use AppendTouchIDs instead.
func TouchIDs() []TouchID {
	return AppendTouchIDs(nil)
}

// TouchPosition returns the position for the touch of the specified ID.
//
// If the touch of the specified ID is not present, TouchPosition returns (0, 0).
//
// TouchPosition is concurrent-safe.
func TouchPosition(id TouchID) (int, int) {
	return inputstate.Get().TouchPosition(ui.TouchID(id))
}

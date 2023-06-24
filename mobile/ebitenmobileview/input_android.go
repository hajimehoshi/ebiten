// Copyright 2016 Hajime Hoshi
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

package ebitenmobileview

import (
	"encoding/hex"
	"hash/crc32"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// https://developer.android.com/reference/android/view/KeyEvent
const (
	keycodeButtonA      = 0x00000060
	keycodeButtonB      = 0x00000061
	keycodeButtonC      = 0x00000062
	keycodeButtonX      = 0x00000063
	keycodeButtonY      = 0x00000064
	keycodeButtonZ      = 0x00000065
	keycodeButtonL1     = 0x00000066
	keycodeButtonR1     = 0x00000067
	keycodeButtonL2     = 0x00000068
	keycodeButtonR2     = 0x00000069
	keycodeButtonThumbl = 0x0000006a
	keycodeButtonThumbr = 0x0000006b
	keycodeButtonStart  = 0x0000006c
	keycodeButtonSelect = 0x0000006d
	keycodeButtonMode   = 0x0000006e
	keycodeButton1      = 0x000000bc
	keycodeButton2      = 0x000000bd
	keycodeButton3      = 0x000000be
	keycodeButton4      = 0x000000bf
	keycodeButton5      = 0x000000c0
	keycodeButton6      = 0x000000c1
	keycodeButton7      = 0x000000c2
	keycodeButton8      = 0x000000c3
	keycodeButton9      = 0x000000c4
	keycodeButton10     = 0x000000c5
	keycodeButton11     = 0x000000c6
	keycodeButton12     = 0x000000c7
	keycodeButton13     = 0x000000c8
	keycodeButton14     = 0x000000c9
	keycodeButton15     = 0x000000ca
	keycodeButton16     = 0x000000cb

	keycodeDpadUp     = 0x00000013
	keycodeDpadDown   = 0x00000014
	keycodeDpadLeft   = 0x00000015
	keycodeDpadRight  = 0x00000016
	keycodeDpadCenter = 0x00000017

	keycodeBack = 0x00000004
	keycodeMenu = 0x00000052
)

// https://developer.android.com/reference/android/view/InputDevice
const (
	sourceKeyboard = 0x00000101
	sourceGamepad  = 0x00000401
	sourceJoystick = 0x01000010
)

// See https://github.com/libsdl-org/SDL/blob/47f2373dc13b66c48bf4024fcdab53cd0bdd59bb/src/joystick/android/SDL_sysjoystick.c#L71-L172
// TODO: This exceeds gamepad.SDLControllerButtonMax. Is that OK?

var androidKeyToSDL = map[int]int{
	keycodeButtonA:      gamepaddb.SDLControllerButtonA,
	keycodeButtonB:      gamepaddb.SDLControllerButtonB,
	keycodeButtonX:      gamepaddb.SDLControllerButtonX,
	keycodeButtonY:      gamepaddb.SDLControllerButtonY,
	keycodeButtonL1:     gamepaddb.SDLControllerButtonLeftShoulder,
	keycodeButtonR1:     gamepaddb.SDLControllerButtonRightShoulder,
	keycodeButtonThumbl: gamepaddb.SDLControllerButtonLeftStick,
	keycodeButtonThumbr: gamepaddb.SDLControllerButtonRightStick,
	keycodeMenu:         gamepaddb.SDLControllerButtonStart,
	keycodeButtonStart:  gamepaddb.SDLControllerButtonStart,
	keycodeBack:         gamepaddb.SDLControllerButtonBack,
	keycodeButtonSelect: gamepaddb.SDLControllerButtonBack,
	keycodeButtonMode:   gamepaddb.SDLControllerButtonGuide,
	keycodeButtonL2:     15,
	keycodeButtonR2:     16,
	keycodeButtonC:      17,
	keycodeButtonZ:      18,
	keycodeDpadUp:       gamepaddb.SDLControllerButtonDpadUp,
	keycodeDpadDown:     gamepaddb.SDLControllerButtonDpadDown,
	keycodeDpadLeft:     gamepaddb.SDLControllerButtonDpadLeft,
	keycodeDpadRight:    gamepaddb.SDLControllerButtonDpadRight,
	keycodeDpadCenter:   gamepaddb.SDLControllerButtonA,
	keycodeButton1:      20,
	keycodeButton2:      21,
	keycodeButton3:      22,
	keycodeButton4:      23,
	keycodeButton5:      24,
	keycodeButton6:      25,
	keycodeButton7:      26,
	keycodeButton8:      27,
	keycodeButton9:      28,
	keycodeButton10:     29,
	keycodeButton11:     30,
	keycodeButton12:     31,
	keycodeButton13:     32,
	keycodeButton14:     33,
	keycodeButton15:     34,
	keycodeButton16:     35,
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	switch action {
	case 0x00, 0x05, 0x02: // ACTION_DOWN, ACTION_POINTER_DOWN, ACTION_MOVE
		touches[ui.TouchID(id)] = position{x, y}
		updateInput(nil)
	case 0x01, 0x06: // ACTION_UP, ACTION_POINTER_UP
		delete(touches, ui.TouchID(id))
		updateInput(nil)
	}
}

func OnKeyDownOnAndroid(keyCode int, unicodeChar int, source int, deviceID int) {
	switch {
	case source&sourceGamepad == sourceGamepad:
		// A gamepad can be detected as a keyboard. Detect the device as a gamepad first.
		if button, ok := androidKeyToSDL[keyCode]; ok {
			gamepad.UpdateAndroidGamepadButton(deviceID, gamepad.Button(button), true)
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToUIKey[keyCode]; ok {
			keys[key] = struct{}{}
		}
		var runes []rune
		if r := rune(unicodeChar); r != 0 && unicode.IsPrint(r) {
			runes = []rune{r}
		}
		updateInput(runes)
	}
}

func OnKeyUpOnAndroid(keyCode int, source int, deviceID int) {
	switch {
	case source&sourceGamepad == sourceGamepad:
		// A gamepad can be detected as a keyboard. Detect the device as a gamepad first.
		if button, ok := androidKeyToSDL[keyCode]; ok {
			gamepad.UpdateAndroidGamepadButton(deviceID, gamepad.Button(button), false)
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToUIKey[keyCode]; ok {
			delete(keys, key)
		}
		updateInput(nil)
	}
}

func OnGamepadAxisChanged(deviceID int, axisID int, value float32) {
	gamepad.UpdateAndroidGamepadAxis(deviceID, axisID, float64(value))
}

func OnGamepadHatChanged(deviceID int, hatID int, xValue, yValue int) {
	gamepad.UpdateAndroidGamepadHat(deviceID, hatID, xValue, yValue)
}

func OnGamepadAdded(deviceID int, name string, axisCount int, hatCount int, descriptor string, vendorID int, productID int, buttonMask int, axisMask int) {
	// This emulates the implementation of Android_AddJoystick.
	// https://github.com/libsdl-org/SDL/blob/0e9560aea22818884921e5e5064953257bfe7fa7/src/joystick/android/SDL_sysjoystick.c#L386
	const SDL_HARDWARE_BUS_BLUETOOTH = 0x05

	var sdlid [16]byte
	sdlid[0] = byte(SDL_HARDWARE_BUS_BLUETOOTH)
	sdlid[1] = byte(SDL_HARDWARE_BUS_BLUETOOTH >> 8)
	if vendorID != 0 && productID != 0 {
		sdlid[4] = byte(vendorID)
		sdlid[5] = byte(vendorID >> 8)
		sdlid[8] = byte(productID)
		sdlid[9] = byte(productID >> 8)
	} else {
		crc := crc32.ChecksumIEEE(([]byte)(descriptor))
		copy(sdlid[4:8], ([]byte)(descriptor))
		sdlid[8] = byte(crc)
		sdlid[9] = byte(crc >> 8)
		sdlid[10] = byte(crc >> 16)
		sdlid[11] = byte(crc >> 24)
	}
	sdlid[12] = byte(buttonMask)
	sdlid[13] = byte(buttonMask >> 8)
	sdlid[14] = byte(axisMask)
	sdlid[15] = byte(axisMask >> 8)

	gamepad.AddAndroidGamepad(deviceID, name, hex.EncodeToString(sdlid[:]), axisCount, hatCount)
}

func OnInputDeviceRemoved(deviceID int) {
	gamepad.RemoveAndroidGamepad(deviceID)
}

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
	"math"
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

	keycodeDpadUp    = 0x00000013
	keycodeDpadDown  = 0x00000014
	keycodeDpadLeft  = 0x00000015
	keycodeDpadRight = 0x00000016
)

// https://developer.android.com/reference/android/view/InputDevice
const (
	sourceKeyboard = 0x00000101
	sourceGamepad  = 0x00000401
	sourceJoystick = 0x01000010
)

// TODO: Can we map these values to the standard gamepad buttons?

var androidKeyToSDLButton = map[int]int{
	keycodeButtonA:      gamepaddb.SDLControllerButtonA,
	keycodeButtonB:      gamepaddb.SDLControllerButtonB,
	keycodeButtonX:      gamepaddb.SDLControllerButtonX,
	keycodeButtonY:      gamepaddb.SDLControllerButtonY,
	keycodeButtonL1:     gamepaddb.SDLControllerButtonLeftShoulder,
	keycodeButtonR1:     gamepaddb.SDLControllerButtonRightShoulder,
	keycodeButtonThumbl: gamepaddb.SDLControllerButtonLeftStick,
	keycodeButtonThumbr: gamepaddb.SDLControllerButtonRightStick,
	keycodeButtonStart:  gamepaddb.SDLControllerButtonStart,
	keycodeButtonSelect: gamepaddb.SDLControllerButtonBack,
	keycodeButtonMode:   gamepaddb.SDLControllerButtonGuide,
}

// Axis constant definitions for joysticks only.
// https://developer.android.com/reference/android/view/MotionEvent
const (
	axisX         = 0x00000000
	axisY         = 0x00000001
	axisZ         = 0x0000000b
	axisRx        = 0x0000000c
	axisRy        = 0x0000000d
	axisRz        = 0x0000000e
	axisHatX      = 0x0000000f
	axisHatY      = 0x00000010
	axisLtrigger  = 0x00000011
	axisRtrigger  = 0x00000012
	axisThrottle  = 0x00000013
	axisRudder    = 0x00000014
	axisWheel     = 0x00000015
	axisGas       = 0x00000016
	axisBrake     = 0x00000017
	axisGeneric1  = 0x00000020
	axisGeneric2  = 0x00000021
	axisGeneric3  = 0x00000022
	axisGeneric4  = 0x00000023
	axisGeneric5  = 0x00000024
	axisGeneric6  = 0x00000025
	axisGeneric7  = 0x00000026
	axisGeneric8  = 0x00000027
	axisGeneric9  = 0x00000028
	axisGeneric10 = 0x00000029
	axisGeneric11 = 0x0000002a
	axisGeneric12 = 0x0000002b
	axisGeneric13 = 0x0000002c
	axisGeneric14 = 0x0000002d
	axisGeneric15 = 0x0000002e
	axisGeneric16 = 0x0000002f
)

var androidAxisToSDLAxis = map[int]int{
	axisX:        gamepaddb.SDLControllerAxisLeftX,
	axisY:        gamepaddb.SDLControllerAxisLeftY,
	axisRx:       gamepaddb.SDLControllerAxisRightX,
	axisRy:       gamepaddb.SDLControllerAxisRightY,
	axisLtrigger: gamepaddb.SDLControllerAxisTriggerLeft,
	axisRtrigger: gamepaddb.SDLControllerAxisTriggerRight,
}

var androidAxisToHatID2 = map[int]int{
	axisHatX: 0,
	axisHatY: 1,
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	switch action {
	case 0x00, 0x05, 0x02: // ACTION_DOWN, ACTION_POINTER_DOWN, ACTION_MOVE
		touches[ui.TouchID(id)] = position{x, y}
		updateInput()
	case 0x01, 0x06: // ACTION_UP, ACTION_POINTER_UP
		delete(touches, ui.TouchID(id))
		updateInput()
	}
}

func OnKeyDownOnAndroid(keyCode int, unicodeChar int, source int, deviceID int) {
	switch {
	case source&sourceGamepad == sourceGamepad:
		// A gamepad can be detected as a keyboard. Detect the device as a gamepad first.
		if button, ok := androidKeyToSDLButton[keyCode]; ok {
			gamepad.UpdateAndroidGamepadButton(deviceID, gamepad.Button(button), true)
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToUIKey[keyCode]; ok {
			keys[key] = struct{}{}
			if r := rune(unicodeChar); r != 0 && unicode.IsPrint(r) {
				runes = []rune{r}
			}
			updateInput()
		}
	}
}

func OnKeyUpOnAndroid(keyCode int, source int, deviceID int) {
	switch {
	case source&sourceGamepad == sourceGamepad:
		// A gamepad can be detected as a keyboard. Detect the device as a gamepad first.
		if button, ok := androidKeyToSDLButton[keyCode]; ok {
			gamepad.UpdateAndroidGamepadButton(deviceID, gamepad.Button(button), false)
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToUIKey[keyCode]; ok {
			delete(keys, key)
			updateInput()
		}
	}
}

func OnGamepadAxesOrHatsChanged(deviceID int, axisID int, value float32) {
	if axis, ok := androidAxisToSDLAxis[axisID]; ok {
		gamepad.UpdateAndroidGamepadAxis(deviceID, axis, float64(value))
		return
	}

	if hid2, ok := androidAxisToHatID2[axisID]; ok {
		const (
			hatUp    = 1
			hatRight = 2
			hatDown  = 4
			hatLeft  = 8
		)
		hatID := hid2 / 2
		var dir gamepad.AndroidHatDirection
		switch hid2 % 2 {
		case 0:
			dir = gamepad.AndroidHatDirectionX
		case 1:
			dir = gamepad.AndroidHatDirectionY
		}
		gamepad.UpdateAndroidGamepadHat(deviceID, hatID, dir, int(math.Round(float64(value))))
		return
	}
}

func OnGamepadAdded(deviceID int, name string, buttonCount int, axisCount int, hatCount int, descriptor string, vendorID int, productID int, buttonMask int, axisMask int) {
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

	gamepad.AddAndroidGamepad(deviceID, name, hex.EncodeToString(sdlid[:]), axisCount, buttonCount, hatCount)
}

func OnInputDeviceRemoved(deviceID int) {
	gamepad.RemoveAndroidGamepad(deviceID)
}

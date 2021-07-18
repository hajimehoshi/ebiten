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

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/uidriver/mobile"
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

var androidKeyToGamepadButton = map[int]driver.GamepadButton{
	keycodeButtonA:      driver.GamepadButton0,
	keycodeButtonB:      driver.GamepadButton1,
	keycodeButtonC:      driver.GamepadButton2,
	keycodeButtonX:      driver.GamepadButton3,
	keycodeButtonY:      driver.GamepadButton4,
	keycodeButtonZ:      driver.GamepadButton5,
	keycodeButtonL1:     driver.GamepadButton6,
	keycodeButtonR1:     driver.GamepadButton7,
	keycodeButtonL2:     driver.GamepadButton8,
	keycodeButtonR2:     driver.GamepadButton9,
	keycodeButtonThumbl: driver.GamepadButton10,
	keycodeButtonThumbr: driver.GamepadButton11,
	keycodeButtonStart:  driver.GamepadButton12,
	keycodeButtonSelect: driver.GamepadButton13,
	keycodeButtonMode:   driver.GamepadButton14,
	keycodeButton1:      driver.GamepadButton15,
	keycodeButton2:      driver.GamepadButton16,
	keycodeButton3:      driver.GamepadButton17,
	keycodeButton4:      driver.GamepadButton18,
	keycodeButton5:      driver.GamepadButton19,
	keycodeButton6:      driver.GamepadButton20,
	keycodeButton7:      driver.GamepadButton21,
	keycodeButton8:      driver.GamepadButton22,
	keycodeButton9:      driver.GamepadButton23,
	keycodeButton10:     driver.GamepadButton24,
	keycodeButton11:     driver.GamepadButton25,
	keycodeButton12:     driver.GamepadButton26,
	keycodeButton13:     driver.GamepadButton27,
	keycodeButton14:     driver.GamepadButton28,
	keycodeButton15:     driver.GamepadButton29,
	keycodeButton16:     driver.GamepadButton30,
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

var androidAxisIDToAxisID = map[int]int{
	axisX:         0,
	axisY:         1,
	axisZ:         2,
	axisRx:        3,
	axisRy:        4,
	axisRz:        5,
	axisHatX:      6,
	axisHatY:      7,
	axisLtrigger:  8,
	axisRtrigger:  9,
	axisThrottle:  10,
	axisRudder:    11,
	axisWheel:     12,
	axisGas:       13,
	axisBrake:     14,
	axisGeneric1:  15,
	axisGeneric2:  16,
	axisGeneric3:  17,
	axisGeneric4:  18,
	axisGeneric5:  19,
	axisGeneric6:  20,
	axisGeneric7:  21,
	axisGeneric8:  22,
	axisGeneric9:  23,
	axisGeneric10: 24,
	axisGeneric11: 25,
	axisGeneric12: 26,
	axisGeneric13: 27,
	axisGeneric14: 28,
	axisGeneric15: 29,
	axisGeneric16: 30,
}

var (
	// deviceIDToGamepadID is a map from Android device IDs to Ebiten gamepad IDs.
	// As convention, Ebiten gamepad IDs start with 0, and many applications depend on this fact.
	deviceIDToGamepadID = map[int]driver.GamepadID{}
)

func gamepadIDFromDeviceID(deviceID int) driver.GamepadID {
	if id, ok := deviceIDToGamepadID[deviceID]; ok {
		return id
	}
	ids := map[driver.GamepadID]struct{}{}
	for _, id := range deviceIDToGamepadID {
		ids[id] = struct{}{}
	}
	for i := driver.GamepadID(0); ; i++ {
		if _, ok := ids[i]; ok {
			continue
		}
		deviceIDToGamepadID[deviceID] = i
		return i
	}
	panic("ebitenmobileview: a gamepad ID cannot be determined")
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	switch action {
	case 0x00, 0x05, 0x02: // ACTION_DOWN, ACTION_POINTER_DOWN, ACTION_MOVE
		touches[driver.TouchID(id)] = position{x, y}
		updateInput()
	case 0x01, 0x06: // ACTION_UP, ACTION_POINTER_UP
		delete(touches, driver.TouchID(id))
		updateInput()
	}
}

func OnKeyDownOnAndroid(keyCode int, unicodeChar int, source int, deviceID int) {
	switch {
	case source&sourceGamepad == sourceGamepad:
		// A gamepad can be detected as a keyboard. Detect the device as a gamepad first.
		if button, ok := androidKeyToGamepadButton[keyCode]; ok {
			id := gamepadIDFromDeviceID(deviceID)
			g, ok := gamepads[id]
			if !ok {
				return
			}
			g.Buttons[button] = true
			updateInput()
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToDriverKey[keyCode]; ok {
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
		if button, ok := androidKeyToGamepadButton[keyCode]; ok {
			id := gamepadIDFromDeviceID(deviceID)
			g, ok := gamepads[id]
			if !ok {
				return
			}
			g.Buttons[button] = false
			updateInput()
		}
	case source&sourceJoystick == sourceJoystick:
		// DPAD keys can come here, but they are also treated as an axis at a motion event. Ignore them.
	case source&sourceKeyboard == sourceKeyboard:
		if key, ok := androidKeyToDriverKey[keyCode]; ok {
			delete(keys, key)
			updateInput()
		}
	}
}

func OnGamepadAxesChanged(deviceID int, axisID int, value float32) {
	did := gamepadIDFromDeviceID(deviceID)
	g, ok := gamepads[did]
	if !ok {
		return
	}
	aid, ok := androidAxisIDToAxisID[axisID]
	if !ok {
		// Unexpected axis value.
		return
	}
	g.Axes[aid] = value
	updateInput()
}

func OnGamepadAdded(deviceID int, name string, buttonNum int, axisNum int, descriptor string, vendorID int, productID int, buttonMask int, axisMask int) {
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

	id := gamepadIDFromDeviceID(deviceID)
	gamepads[id] = &mobile.Gamepad{
		ID:        id,
		SDLID:     hex.EncodeToString(sdlid[:]),
		Name:      name,
		ButtonNum: buttonNum,
		AxisNum:   axisNum,
	}
}

func OnInputDeviceRemoved(deviceID int) {
	if id, ok := deviceIDToGamepadID[deviceID]; ok {
		delete(gamepads, id)
		delete(deviceIDToGamepadID, deviceID)
	}
}

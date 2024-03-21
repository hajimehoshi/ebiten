// Copyright 2022 The Ebitengine Authors
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

package gamepad

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func standardButtonToGamepadInputGamepadButton(b gamepaddb.StandardButton) (_GameInputGamepadButtons, bool) {
	switch b {
	case gamepaddb.StandardButtonRightBottom:
		return _GameInputGamepadA, true
	case gamepaddb.StandardButtonRightRight:
		return _GameInputGamepadB, true
	case gamepaddb.StandardButtonRightLeft:
		return _GameInputGamepadX, true
	case gamepaddb.StandardButtonRightTop:
		return _GameInputGamepadY, true
	case gamepaddb.StandardButtonFrontTopLeft:
		return _GameInputGamepadLeftShoulder, true
	case gamepaddb.StandardButtonFrontTopRight:
		return _GameInputGamepadRightShoulder, true
	case gamepaddb.StandardButtonFrontBottomLeft:
		return 0, false // Use leftTrigger instead.
	case gamepaddb.StandardButtonFrontBottomRight:
		return 0, false // Use rightTrigger instead.
	case gamepaddb.StandardButtonCenterLeft:
		return _GameInputGamepadView, true
	case gamepaddb.StandardButtonCenterRight:
		return _GameInputGamepadMenu, true
	case gamepaddb.StandardButtonLeftStick:
		return _GameInputGamepadLeftThumbstick, true
	case gamepaddb.StandardButtonRightStick:
		return _GameInputGamepadRightThumbstick, true
	case gamepaddb.StandardButtonLeftTop:
		return _GameInputGamepadDPadUp, true
	case gamepaddb.StandardButtonLeftBottom:
		return _GameInputGamepadDPadDown, true
	case gamepaddb.StandardButtonLeftLeft:
		return _GameInputGamepadDPadLeft, true
	case gamepaddb.StandardButtonLeftRight:
		return _GameInputGamepadDPadRight, true
	case gamepaddb.StandardButtonCenterCenter:
		return 0, false
	}
	return 0, false
}

type nativeGamepadsXbox struct {
	gameInput         *_IGameInput
	deviceCallbackPtr uintptr
	token             _GameInputCallbackToken
}

func (n *nativeGamepadsXbox) init(gamepads *gamepads) error {
	g, err := _GameInputCreate()
	if err != nil {
		return err
	}

	n.gameInput = g
	n.deviceCallbackPtr = windows.NewCallbackCDecl(n.deviceCallback)

	if err := n.gameInput.RegisterDeviceCallback(
		nil,
		_GameInputKindGamepad,
		_GameInputDeviceConnected,
		_GameInputBlockingEnumeration,
		unsafe.Pointer(gamepads),
		n.deviceCallbackPtr,
		&n.token,
	); err != nil {
		return err
	}
	return nil
}

func (n *nativeGamepadsXbox) update(gamepads *gamepads) error {
	return nil
}

func (n *nativeGamepadsXbox) deviceCallback(callbackToken _GameInputCallbackToken, context unsafe.Pointer, device *_IGameInputDevice, timestamp uint64, currentStatus _GameInputDeviceStatus, previousStatus _GameInputDeviceStatus) uintptr {
	gps := (*gamepads)(context)

	// Connected.
	if currentStatus&_GameInputDeviceConnected != 0 {
		// TODO: Give a good name and a SDL ID.
		gp := gps.add("", "00000000000000000000000000000000")
		gp.native = &nativeGamepadXbox{
			gameInputDevice: device,
		}
		return 0
	}

	// Disconnected.
	gps.remove(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadXbox).gameInputDevice == device
	})

	return 0
}

type nativeGamepadXbox struct {
	gameInputDevice *_IGameInputDevice
	state           _GameInputGamepadState

	vib    bool
	vibEnd time.Time
}

func (n *nativeGamepadXbox) update(gamepads *gamepads) error {
	gameInput := gamepads.native.(*nativeGamepadsXbox).gameInput
	r, err := gameInput.GetCurrentReading(_GameInputKindGamepad, n.gameInputDevice)
	if err != nil {
		return err
	}
	defer r.Release()

	state, ok := r.GetGamepadState()
	if !ok {
		n.state = _GameInputGamepadState{}
		return nil
	}
	n.state = state

	if n.vib && time.Now().Sub(n.vibEnd) >= 0 {
		n.gameInputDevice.SetRumbleState(&_GameInputRumbleParams{
			lowFrequency:  0,
			highFrequency: 0,
		}, 0)
		n.vib = false
	}

	return nil
}

func (n *nativeGamepadXbox) hasOwnStandardLayoutMapping() bool {
	return true
}

func (n *nativeGamepadXbox) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	switch axis {
	case gamepaddb.StandardAxisLeftStickHorizontal,
		gamepaddb.StandardAxisLeftStickVertical,
		gamepaddb.StandardAxisRightStickHorizontal,
		gamepaddb.StandardAxisRightStickVertical:
		return axisMappingInput{g: n, axis: int(axis)}
	}
	return nil
}

func (n *nativeGamepadXbox) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	switch button {
	case gamepaddb.StandardButtonFrontBottomLeft,
		gamepaddb.StandardButtonFrontBottomRight:
		return buttonMappingInput{g: n, button: int(button)}
	}
	if _, ok := standardButtonToGamepadInputGamepadButton(button); !ok {
		return nil
	}
	return buttonMappingInput{g: n, button: int(button)}
}

func (n *nativeGamepadXbox) axisCount() int {
	return int(gamepaddb.StandardAxisMax) + 1
}

func (n *nativeGamepadXbox) buttonCount() int {
	return int(gamepaddb.StandardButtonMax) + 1
}

func (n *nativeGamepadXbox) hatCount() int {
	return 0
}

func (g *nativeGamepadXbox) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (n *nativeGamepadXbox) axisValue(axis int) float64 {
	switch gamepaddb.StandardAxis(axis) {
	case gamepaddb.StandardAxisLeftStickHorizontal:
		return float64(n.state.leftThumbstickX)
	case gamepaddb.StandardAxisLeftStickVertical:
		return -float64(n.state.leftThumbstickY)
	case gamepaddb.StandardAxisRightStickHorizontal:
		return float64(n.state.rightThumbstickX)
	case gamepaddb.StandardAxisRightStickVertical:
		return -float64(n.state.rightThumbstickY)
	}
	return 0
}

func (n *nativeGamepadXbox) buttonValue(button int) float64 {
	switch gamepaddb.StandardButton(button) {
	case gamepaddb.StandardButtonFrontBottomLeft:
		return float64(n.state.leftTrigger)
	case gamepaddb.StandardButtonFrontBottomRight:
		return float64(n.state.rightTrigger)
	}
	b, ok := standardButtonToGamepadInputGamepadButton(gamepaddb.StandardButton(button))
	if !ok {
		return 0
	}
	if n.state.buttons&b != 0 {
		return 1
	}
	return 0
}

func (n *nativeGamepadXbox) isButtonPressed(button int) bool {
	switch gamepaddb.StandardButton(button) {
	case gamepaddb.StandardButtonFrontBottomLeft:
		return n.state.leftTrigger > gamepaddb.ButtonPressedThreshold
	case gamepaddb.StandardButtonFrontBottomRight:
		return n.state.rightTrigger > gamepaddb.ButtonPressedThreshold
	}

	b, ok := standardButtonToGamepadInputGamepadButton(gamepaddb.StandardButton(button))
	if !ok {
		return false
	}
	if n.state.buttons&b != 0 {
		return true
	}
	return false
}

func (n *nativeGamepadXbox) hatState(hat int) int {
	return 0
}

func (n *nativeGamepadXbox) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		n.vib = false
		n.gameInputDevice.SetRumbleState(&_GameInputRumbleParams{
			lowFrequency:  0,
			highFrequency: 0,
		}, 0)
		return
	}
	n.vib = true
	n.vibEnd = time.Now().Add(duration)
	n.gameInputDevice.SetRumbleState(&_GameInputRumbleParams{
		lowFrequency:  float32(strongMagnitude),
		highFrequency: float32(weakMagnitude),
	}, 0)
}

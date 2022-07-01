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

//go:build !ebitencbackend
// +build !ebitencbackend

package gamepad

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

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
		_GameInputKindAny,
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
}

func (n *nativeGamepadXbox) update(gamepads *gamepads) error {
	return nil
}

func (n *nativeGamepadXbox) hasOwnStandardLayoutMapping() bool {
	return true
}

func (n *nativeGamepadXbox) axisCount() int {
	return 0
}

func (n *nativeGamepadXbox) buttonCount() int {
	return 0
}

func (n *nativeGamepadXbox) hatCount() int {
	return 0
}

func (n *nativeGamepadXbox) axisValue(axis int) float64 {
	return 0
}

func (n *nativeGamepadXbox) buttonValue(button int) float64 {
	return 0
}

func (n *nativeGamepadXbox) isButtonPressed(button int) bool {
	return false
}

func (n *nativeGamepadXbox) hatState(hat int) int {
	return 0
}

func (n *nativeGamepadXbox) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
}

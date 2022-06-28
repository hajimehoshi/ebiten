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
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeGamepadsXbox struct {
	gameInput         *_IGameInput
	deviceCallbackPtr uintptr
	token             _GameInputCallbackToken
}

func xboxDeviceCallback(
	callbackToken _GameInputCallbackToken,
	context unsafe.Pointer,
	device *_IGameInputDevice,
	timestamp uint64,
	currentStatus _GameInputDeviceStatus,
	previousStatus _GameInputDeviceStatus) uintptr {
	// TODO: Implement this.
	return 0
}

func (n *nativeGamepadsXbox) init(gamepads *gamepads) error {
	g, err := _GameInputCreate()
	if err != nil {
		return err
	}

	n.gameInput = g
	n.deviceCallbackPtr = windows.NewCallbackCDecl(xboxDeviceCallback)

	if err := n.gameInput.RegisterDeviceCallback(
		nil,
		_GameInputKindGamepad,
		_GameInputDeviceConnected,
		_GameInputBlockingEnumeration,
		nil,
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

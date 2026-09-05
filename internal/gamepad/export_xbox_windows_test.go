// Copyright 2026 The Ebitengine Authors
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

type (
	Gamepads         = gamepads
	XboxGamepads     = nativeGamepadsXbox
	IGameInputDevice = _IGameInputDevice
)

// DeviceCallback reports device as connected or disconnected, as GameInput's device callback does.
func (n *nativeGamepadsXbox) DeviceCallback(device *_IGameInputDevice, connected bool) {
	var status _GameInputDeviceStatus
	if connected {
		status = _GameInputDeviceConnected
	}
	n.deviceCallback(0, nil, device, 0, status, 0)
}

// Update applies the queued device events to gamepads.
func (n *nativeGamepadsXbox) Update(gamepads *gamepads) error {
	return n.update(gamepads)
}

// AppendXboxDevices appends the GameInput device of each gamepad registered in g.
func (g *gamepads) AppendXboxDevices(devices []*_IGameInputDevice) []*_IGameInputDevice {
	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		devices = append(devices, gp.native.(*nativeGamepadXbox).gameInputDevice)
	}
	return devices
}

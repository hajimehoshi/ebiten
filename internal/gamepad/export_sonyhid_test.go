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

type SonyModel = sonyModel

const (
	SonyModelNone       = sonyModelNone
	SonyModelDualShock4 = sonyModelDualShock4
	SonyModelDualSense  = sonyModelDualSense
)

const (
	Dualshock4OutputReportSizeUSB = dualshock4OutputReportSizeUSB
	Dualshock4OutputReportSizeBT  = dualshock4OutputReportSizeBT
	DualsenseOutputReportSizeUSB  = dualsenseOutputReportSizeUSB
	DualsenseOutputReportSizeBT   = dualsenseOutputReportSizeBT
	Dualshock4InputReportSizeUSB  = dualshock4InputReportSizeUSB
	DualsenseInputReportSizeUSB   = dualsenseInputReportSizeUSB
	SonySimpleInputReportSizeBT   = sonySimpleInputReportSizeBT
	Dualshock4InputReportSizeBT   = dualshock4InputReportSizeBT
	DualsenseInputReportSizeBT    = dualsenseInputReportSizeBT
)

type SonyInputState struct {
	LX, LY, RX, RY byte
	L2, R2         byte
	Hat            byte
	Buttons        uint16
}

func SonyInputStateFromReport(model sonyModel, bt bool, report []byte) (SonyInputState, bool) {
	s, ok := sonyInputStateFromReport(model, bt, report)
	return SonyInputState{
		LX: s.lx, LY: s.ly, RX: s.rx, RY: s.ry,
		L2: s.l2, R2: s.r2,
		Hat:     s.hat,
		Buttons: s.buttons,
	}, ok
}

var (
	SonyModelFromIDs              = sonyModelFromIDs
	BluetoothFromDeviceInstanceID = bluetoothFromDeviceInstanceID
	SonyOutputReportSize          = sonyOutputReportSize
	SonyInputReportSize           = sonyInputReportSize
	SonyRumbleByte                = sonyRumbleByte
	SonyBTCRC                     = sonyBTCRC
	Dualshock4RumbleReportUSB     = dualshock4RumbleReportUSB
	Dualshock4RumbleReportBT      = dualshock4RumbleReportBT
	DualsenseRumbleReportUSB      = dualsenseRumbleReportUSB
	DualsenseRumbleReportBT       = dualsenseRumbleReportBT
)

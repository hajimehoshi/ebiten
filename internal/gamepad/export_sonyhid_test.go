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
)

var (
	SonyModelFromIDs              = sonyModelFromIDs
	BluetoothFromDeviceInstanceID = bluetoothFromDeviceInstanceID
	SonyOutputReportSize          = sonyOutputReportSize
	SonyRumbleByte                = sonyRumbleByte
	SonyBTCRC                     = sonyBTCRC
	Dualshock4RumbleReportUSB     = dualshock4RumbleReportUSB
	Dualshock4RumbleReportBT      = dualshock4RumbleReportBT
	DualsenseRumbleReportUSB      = dualsenseRumbleReportUSB
	DualsenseRumbleReportBT       = dualsenseRumbleReportBT
)

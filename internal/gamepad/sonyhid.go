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

import (
	"encoding/binary"
	"hash/crc32"
	"strings"
)

// This file builds the vendor-specific HID output reports that control rumble
// on PlayStation controllers. The reports differ by controller model and by
// transport (USB or Bluetooth), and Bluetooth reports carry a trailing CRC32.
// This file is platform-independent so the report layouts can be unit tested
// anywhere; writing the reports to a device is platform-specific.

type sonyModel int

const (
	sonyModelNone sonyModel = iota
	sonyModelDualShock4
	sonyModelDualSense
)

const sonyVendorID = 0x054c

// sonyModelFromIDs reports the PlayStation controller model for a USB
// vendor/product ID pair, or sonyModelNone if the IDs are not a known
// PlayStation controller.
func sonyModelFromIDs(vid, pid uint16) sonyModel {
	if vid != sonyVendorID {
		return sonyModelNone
	}
	switch pid {
	case 0x05c4, 0x09cc:
		return sonyModelDualShock4
	case 0x0ce6, 0x0df2:
		return sonyModelDualSense
	}
	return sonyModelNone
}

// bluetoothFromDeviceInstanceID reports whether a Windows device instance ID
// names a device enumerated over Bluetooth or over USB, from the ID's leading
// enumerator name. ok is false for any other enumerator.
//
// A device instance ID begins with the enumerator's device ID, so USB devices
// use "USB\..." and Bluetooth devices use "BTHENUM\{ServiceGUID}...". See
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/device-instance-ids,
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/standard-usb-identifiers, and
// https://learn.microsoft.com/en-us/windows-hardware/drivers/bluetooth/installing-a-bluetooth-device.
func bluetoothFromDeviceInstanceID(id string) (bt, ok bool) {
	hasPrefixFold := func(s, prefix string) bool {
		return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
	}
	switch {
	case hasPrefixFold(id, `BTHENUM\`):
		return true, true
	case hasPrefixFold(id, `USB\`):
		return false, true
	}
	return false, false
}

// Report sizes include the leading report ID byte.
const (
	dualshock4ReportSizeUSB = 32
	dualshock4ReportSizeBT  = 78
	dualsenseReportSizeUSB  = 48
	dualsenseReportSizeBT   = 78
)

// sonyReportSize returns the output report size for a model and transport,
// or 0 for an unknown model.
func sonyReportSize(model sonyModel, bt bool) int {
	switch model {
	case sonyModelDualShock4:
		if bt {
			return dualshock4ReportSizeBT
		}
		return dualshock4ReportSizeUSB
	case sonyModelDualSense:
		if bt {
			return dualsenseReportSizeBT
		}
		return dualsenseReportSizeUSB
	}
	return 0
}

// sonyRumbleByte converts a magnitude in the range 0 to 1 to a motor value.
// Out-of-range values are clamped, and NaN is treated as 0, since converting
// such values to an integer is implementation-defined.
func sonyRumbleByte(magnitude float64) byte {
	if !(magnitude > 0) {
		return 0
	}
	if magnitude > 1 {
		return 0xff
	}
	return byte(magnitude * 0xff)
}

var sonyCRCTable = crc32.MakeTable(crc32.IEEE)

// sonyBTCRC computes the CRC32 carried in the last 4 bytes of Bluetooth
// output reports: an IEEE CRC32 over a 0xa2 prefix byte (the HID output
// transaction header) followed by the report bytes before the CRC itself.
func sonyBTCRC(data []byte) uint32 {
	crc := crc32.Update(0, sonyCRCTable, []byte{0xa2})
	return crc32.Update(crc, sonyCRCTable, data)
}

// putSonyBTCRC writes the CRC over everything before the last 4 bytes of
// report into its last 4 bytes.
func putSonyBTCRC(report []byte) {
	n := len(report) - 4
	binary.LittleEndian.PutUint32(report[n:], sonyBTCRC(report[:n]))
}

// dualshock4RumbleReportUSB builds output report 0x05.
//
// Only the rumble-valid flag is set so the light bar and flash state are left
// as they are.
func dualshock4RumbleReportUSB(strong, weak byte) []byte {
	r := make([]byte, dualshock4ReportSizeUSB)
	r[0] = 0x05
	r[1] = 0x01 // Rumble fields are valid.
	r[4] = weak
	r[5] = strong
	return r
}

// dualshock4RumbleReportBT builds output report 0x11 with its CRC trailer.
func dualshock4RumbleReportBT(strong, weak byte) []byte {
	r := make([]byte, dualshock4ReportSizeBT)
	r[0] = 0x11
	r[1] = 0xc0 // HID output with a CRC32 trailer.
	r[3] = 0x01 // Rumble fields are valid.
	r[6] = weak
	r[7] = strong
	putSonyBTCRC(r)
	return r
}

// dualsenseSetCommon fills the common output payload shared by the USB and
// Bluetooth report framings.
//
// The compatible-vibration and haptics-select flags route the motor values to
// the rumble emulation; no other state (LEDs, triggers, audio) is marked
// valid.
func dualsenseSetCommon(p []byte, strong, weak byte) {
	p[0] = 0x03 // Compatible vibration + haptics select.
	p[2] = weak
	p[3] = strong
}

// dualsenseRumbleReportUSB builds output report 0x02.
func dualsenseRumbleReportUSB(strong, weak byte) []byte {
	r := make([]byte, dualsenseReportSizeUSB)
	r[0] = 0x02
	dualsenseSetCommon(r[1:], strong, weak)
	return r
}

// dualsenseRumbleReportBT builds output report 0x31 with its CRC trailer.
// seq is a per-device counter; only its low 4 bits are used.
func dualsenseRumbleReportBT(seq, strong, weak byte) []byte {
	r := make([]byte, dualsenseReportSizeBT)
	r[0] = 0x31
	r[1] = (seq & 0x0f) << 4
	r[2] = 0x10 // Output report tag.
	dualsenseSetCommon(r[3:], strong, weak)
	putSonyBTCRC(r)
	return r
}

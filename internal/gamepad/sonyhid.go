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
// on PlayStation controllers, and decodes the input reports the controllers
// send over Bluetooth. The reports differ by controller model and by
// transport (USB or Bluetooth), and Bluetooth reports carry a trailing CRC32.
// This file is platform-independent so the report layouts can be unit tested
// anywhere; exchanging the reports with a device is platform-specific.

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

// Report sizes include the leading report ID byte. That the Bluetooth sizes
// match across models and directions is a coincidence.
const (
	dualShock4OutputReportSizeUSB = 32
	dualShock4OutputReportSizeBT  = 78
	dualSenseOutputReportSizeUSB  = 48
	dualSenseOutputReportSizeBT   = 78

	dualShock4InputReportSizeUSB = 64 // Report 0x01.
	dualSenseInputReportSizeUSB  = 64 // Report 0x01.

	// sonySimpleInputReportSizeBT is the size of input report 0x01, which
	// both models send over Bluetooth until they receive an output report.
	// Receiving one switches a controller to its full input report for the
	// rest of the session.
	sonySimpleInputReportSizeBT = 10
	dualShock4InputReportSizeBT = 78 // Report 0x11.
	dualSenseInputReportSizeBT  = 78 // Report 0x31.
)

// sonyInputReportSize returns the size of the full input report a model sends
// over a transport, or 0 for an unknown model.
func sonyInputReportSize(model sonyModel, bt bool) int {
	switch model {
	case sonyModelDualShock4:
		if bt {
			return dualShock4InputReportSizeBT
		}
		return dualShock4InputReportSizeUSB
	case sonyModelDualSense:
		if bt {
			return dualSenseInputReportSizeBT
		}
		return dualSenseInputReportSizeUSB
	}
	return 0
}

// sonyOutputReportSize returns the output report size for a model and
// transport, or 0 for an unknown model.
func sonyOutputReportSize(model sonyModel, bt bool) int {
	switch model {
	case sonyModelDualShock4:
		if bt {
			return dualShock4OutputReportSizeBT
		}
		return dualShock4OutputReportSizeUSB
	case sonyModelDualSense:
		if bt {
			return dualSenseOutputReportSizeBT
		}
		return dualSenseOutputReportSizeUSB
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

// The Bluetooth HID transaction headers that prefix the data a CRC covers.
// Input reports come from the device and output reports go to it.
const (
	sonyBTInputHeader  = 0xa1
	sonyBTOutputHeader = 0xa2
)

// sonyBTCRC computes the CRC32 carried in the last 4 bytes of full Bluetooth
// reports: an IEEE CRC32 over the report's transaction header byte followed
// by the report bytes before the CRC itself.
func sonyBTCRC(header byte, data []byte) uint32 {
	crc := crc32.Update(0, sonyCRCTable, []byte{header})
	return crc32.Update(crc, sonyCRCTable, data)
}

// putSonyBTCRC writes the CRC over everything before the last 4 bytes of an
// output report into its last 4 bytes.
func putSonyBTCRC(report []byte) {
	n := len(report) - 4
	binary.LittleEndian.PutUint32(report[n:], sonyBTCRC(sonyBTOutputHeader, report[:n]))
}

// sonyBTInputCRCValid reports whether the last 4 bytes of an input report
// hold the CRC over everything before them. report must be at least 4 bytes.
func sonyBTInputCRCValid(report []byte) bool {
	n := len(report) - 4
	return binary.LittleEndian.Uint32(report[n:]) == sonyBTCRC(sonyBTInputHeader, report[:n])
}

// dualShock4RumbleReportUSB builds output report 0x05.
//
// Only the rumble-valid flag is set so the light bar and flash state are left
// as they are.
func dualShock4RumbleReportUSB(strong, weak byte) []byte {
	r := make([]byte, dualShock4OutputReportSizeUSB)
	r[0] = 0x05
	r[1] = 0x01 // Rumble fields are valid.
	r[4] = weak
	r[5] = strong
	return r
}

// dualShock4RumbleReportBT builds output report 0x11 with its CRC trailer.
func dualShock4RumbleReportBT(strong, weak byte) []byte {
	r := make([]byte, dualShock4OutputReportSizeBT)
	r[0] = 0x11
	r[1] = 0xc0 // HID output with a CRC32 trailer.
	r[3] = 0x01 // Rumble fields are valid.
	r[6] = weak
	r[7] = strong
	putSonyBTCRC(r)
	return r
}

// setDualSenseCommonOutput fills the common output payload shared by the USB and
// Bluetooth report framings.
//
// The compatible-vibration and haptics-select flags route the motor values to
// the rumble emulation; no other state (LEDs, triggers, audio) is marked
// valid.
func setDualSenseCommonOutput(p []byte, strong, weak byte) {
	p[0] = 0x03 // Compatible vibration + haptics select.
	p[2] = weak
	p[3] = strong
}

// dualSenseRumbleReportUSB builds output report 0x02.
func dualSenseRumbleReportUSB(strong, weak byte) []byte {
	r := make([]byte, dualSenseOutputReportSizeUSB)
	r[0] = 0x02
	setDualSenseCommonOutput(r[1:], strong, weak)
	return r
}

// dualSenseRumbleReportBT builds output report 0x31 with its CRC trailer.
// seq is a per-device counter; only its low 4 bits are used.
func dualSenseRumbleReportBT(seq, strong, weak byte) []byte {
	r := make([]byte, dualSenseOutputReportSizeBT)
	r[0] = 0x31
	r[1] = (seq & 0x0f) << 4
	r[2] = 0x10 // Output report tag.
	setDualSenseCommonOutput(r[3:], strong, weak)
	putSonyBTCRC(r)
	return r
}

// sonyInputState is the controller state carried by an input report: the
// stick and trigger positions, the D-pad direction, and the buttons.
//
// The button numbering follows the order in which the controllers' HID
// descriptors declare the button usages, which is also the order DirectInput
// enumerates the buttons in, so the state can stand in for a DirectInput
// state.
type sonyInputState struct {
	// lx, ly, rx, and ry are the stick positions. 0 is left or up, and 0x80
	// is the center.
	lx, ly, rx, ry byte

	// l2 and r2 are the trigger positions. 0 is released.
	l2, r2 byte

	// hat is the D-pad direction: 0 is up, and each step of 1 turns 45
	// degrees clockwise. 8 or above is centered.
	hat byte

	// buttons has bit i set while button i+1 is pressed:
	//
	//	0 Square, 1 Cross, 2 Circle, 3 Triangle, 4 L1, 5 R1, 6 L2, 7 R2,
	//	8 Share (Create), 9 Options, 10 L3, 11 R3, 12 PS, 13 Touchpad,
	//	14 Mute (DualSense only).
	buttons uint16
}

// sonyNeutralInputState is the state of a controller at rest: sticks centered,
// triggers released, D-pad centered, and no buttons pressed. It stands in
// until the first input report is decoded.
var sonyNeutralInputState = sonyInputState{
	lx: 0x80, ly: 0x80, rx: 0x80, ry: 0x80,
	hat: 8,
}

// sonyDecodeButtons decodes the three button bytes shared by all the report
// layouts. b0 carries the D-pad direction in its low nibble and the face
// buttons in its high nibble; b1 carries the shoulders, triggers, Share,
// Options, and stick buttons; b2 carries PS, Touchpad, and, on the DualSense,
// Mute in its low bits. mask selects the bits of b2 that are buttons: the
// remaining bits are a counter on the DualShock 4 and reserved on the
// DualSense.
func sonyDecodeButtons(b0, b1, b2, mask byte) (hat byte, buttons uint16) {
	return b0 & 0x0f, uint16(b0>>4) | uint16(b1)<<4 | uint16(b2&mask)<<12
}

// sonyInputStateDS4Layout decodes the state payload of a DualShock 4 input
// report. p must be at least 9 bytes. The DualSense uses the same layout in
// its simplified Bluetooth report, without a Mute bit.
func sonyInputStateDS4Layout(p []byte) sonyInputState {
	hat, buttons := sonyDecodeButtons(p[4], p[5], p[6], 0x03)
	return sonyInputState{
		lx: p[0], ly: p[1], rx: p[2], ry: p[3],
		l2: p[7], r2: p[8],
		hat:     hat,
		buttons: buttons,
	}
}

// sonyInputStateDualSenseLayout decodes the state payload of a DualSense full
// input report. p must be at least 10 bytes.
func sonyInputStateDualSenseLayout(p []byte) sonyInputState {
	// p[6] is a counter.
	hat, buttons := sonyDecodeButtons(p[7], p[8], p[9], 0x07)
	return sonyInputState{
		lx: p[0], ly: p[1], rx: p[2], ry: p[3],
		l2: p[4], r2: p[5],
		hat:     hat,
		buttons: buttons,
	}
}

// sonyInputStateFromReport decodes an input report received from a controller
// over a transport. ok is false if the report is not a state report the model
// sends over that transport, is shorter than that report, or fails the
// validation below; the caller keeps its previous state then. A report may be
// longer than the state report when the host pads reports to the device's
// maximum report length; the trailing bytes are ignored.
//
// Over USB both models send report 0x01 with their own layout at an offset of
// 1. Over Bluetooth both send the simplified report 0x01, in the DualShock 4
// layout, until the first output report. After that, the DualShock 4 sends
// report 0x11 with its layout at an offset of 3, and the DualSense sends
// report 0x31 with its layout at an offset of 2. The transport is needed to
// tell the DualSense's two report 0x01 layouts apart.
//
// The full Bluetooth reports end in a CRC over the bytes before it, and a
// report whose CRC does not match is corrupted. The DualShock 4's report 0x11
// also carries controller state only when bit 7 of its byte 1 is set.
func sonyInputStateFromReport(model sonyModel, bt bool, report []byte) (state sonyInputState, ok bool) {
	if len(report) == 0 {
		return sonyInputState{}, false
	}
	var offset, size int
	var decode func([]byte) sonyInputState
	var crc bool
	switch {
	case report[0] == 0x01 && !bt && model == sonyModelDualShock4:
		offset, size, decode = 1, dualShock4InputReportSizeUSB, sonyInputStateDS4Layout
	case report[0] == 0x01 && !bt && model == sonyModelDualSense:
		offset, size, decode = 1, dualSenseInputReportSizeUSB, sonyInputStateDualSenseLayout
	case report[0] == 0x01 && bt && (model == sonyModelDualShock4 || model == sonyModelDualSense):
		offset, size, decode = 1, sonySimpleInputReportSizeBT, sonyInputStateDS4Layout
	case report[0] == 0x11 && bt && model == sonyModelDualShock4:
		offset, size, decode, crc = 3, dualShock4InputReportSizeBT, sonyInputStateDS4Layout, true
	case report[0] == 0x31 && bt && model == sonyModelDualSense:
		offset, size, decode, crc = 2, dualSenseInputReportSizeBT, sonyInputStateDualSenseLayout, true
	default:
		return sonyInputState{}, false
	}
	if len(report) < size {
		return sonyInputState{}, false
	}
	report = report[:size]
	if crc && !sonyBTInputCRCValid(report) {
		return sonyInputState{}, false
	}
	if report[0] == 0x11 && report[1]&0x80 == 0 {
		return sonyInputState{}, false
	}
	return decode(report[offset:]), true
}

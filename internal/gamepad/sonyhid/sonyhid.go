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

package sonyhid

import (
	"encoding/binary"
	"hash/crc32"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

type model int

const (
	modelNone model = iota
	modelDualShock4
	modelDualSense
)

const vendorID = 0x054c

// modelFromIDs reports the PlayStation controller model for a USB
// vendor/product ID pair, or modelNone if the IDs are not a known
// PlayStation controller.
func modelFromIDs(vid, pid uint16) model {
	if vid != vendorID {
		return modelNone
	}
	switch pid {
	case 0x05c4, 0x09cc:
		return modelDualShock4
	case 0x0ce6, 0x0df2:
		return modelDualSense
	}
	return modelNone
}

// bluetoothFromDeviceInstanceID reports whether a Windows device instance ID
// names a device enumerated over Bluetooth or over USB, from the ID's leading
// enumerator name. ok is false for any other enumerator.
//
// A device instance ID begins with the name of the enumerator, so USB devices
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

	// simpleInputReportSizeBT is the size of input report 0x01, which
	// both models send over Bluetooth until they receive an output report.
	// Receiving one switches a controller to its full input report for the
	// rest of the session.
	simpleInputReportSizeBT     = 10
	dualShock4InputReportSizeBT = 78 // Report 0x11.
	dualSenseInputReportSizeBT  = 78 // Report 0x31.
)

// inputReportSize returns the size of the full input report a model sends
// over a transport, or 0 for an unknown model.
func inputReportSize(model model, bt bool) int {
	switch model {
	case modelDualShock4:
		if bt {
			return dualShock4InputReportSizeBT
		}
		return dualShock4InputReportSizeUSB
	case modelDualSense:
		if bt {
			return dualSenseInputReportSizeBT
		}
		return dualSenseInputReportSizeUSB
	}
	return 0
}

// outputReportSize returns the output report size for a model and
// transport, or 0 for an unknown model.
func outputReportSize(model model, bt bool) int {
	switch model {
	case modelDualShock4:
		if bt {
			return dualShock4OutputReportSizeBT
		}
		return dualShock4OutputReportSizeUSB
	case modelDualSense:
		if bt {
			return dualSenseOutputReportSizeBT
		}
		return dualSenseOutputReportSizeUSB
	}
	return 0
}

// rumbleByte converts a magnitude in the range 0 to 1 to a motor value.
// Out-of-range values are clamped and NaN is treated as 0.
func rumbleByte(magnitude float64) byte {
	// Converting an out-of-range or NaN value to an integer is implementation-defined,
	// so such values must be rejected before the conversion.
	return byte(mathutil.Clamp01(magnitude) * 0xff)
}

var crcTable = crc32.MakeTable(crc32.IEEE)

// The Bluetooth HID transaction headers that prefix the data a CRC covers.
// Input reports come from the device and output reports go to it.
const (
	btInputHeader  = 0xa1
	btOutputHeader = 0xa2
)

// btCRC computes the CRC32 carried in the last 4 bytes of full Bluetooth
// reports: an IEEE CRC32 over the report's transaction header byte followed
// by the report bytes before the CRC itself.
func btCRC(header byte, data []byte) uint32 {
	crc := crc32.Update(0, crcTable, []byte{header})
	return crc32.Update(crc, crcTable, data)
}

// putBTCRC writes the CRC over everything before the last 4 bytes of an
// output report into its last 4 bytes.
func putBTCRC(report []byte) {
	n := len(report) - 4
	binary.LittleEndian.PutUint32(report[n:], btCRC(btOutputHeader, report[:n]))
}

// btInputCRCValid reports whether the last 4 bytes of an input report
// hold the CRC over everything before them. report must be at least 4 bytes.
func btInputCRCValid(report []byte) bool {
	n := len(report) - 4
	return binary.LittleEndian.Uint32(report[n:]) == btCRC(btInputHeader, report[:n])
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
	putBTCRC(r)
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
	putBTCRC(r)
	return r
}

// TouchCount is the number of fingers the touchpad tracks at once.
const TouchCount = 2

// Touch is one of the fingers the touchpad tracks.
type Touch struct {
	// Active reports that the finger is on the touchpad.
	Active bool

	// ID is the controller's 7-bit contact counter, which differs between one
	// contact and the next in the same slot.
	ID byte

	// X and Y are the finger's position, each in -1..1 with (-1, -1) at the
	// top left of the touchpad.
	X, Y float64
}

// touchpadSize returns the touchpad's grid in the units of the reports, or
// 0, 0 for an unknown model.
func touchpadSize(model model) (w, h int) {
	switch model {
	case modelDualShock4:
		return 1920, 942
	case modelDualSense:
		return 1920, 1080
	}
	return 0, 0
}

// touchCoord maps a grid coordinate to -1..1 over a grid of n units, clamping
// values past the grid.
func touchCoord(v, n int) float64 {
	if n < 2 {
		return 0
	}
	return mathutil.Clamp01(float64(v)/float64(n-1))*2 - 1
}

// touchFromRecord decodes a 4-byte touch record of a touchpad w by h units:
// bit 7 of the first byte is set while no finger is down and its low 7 bits
// are the contact counter; the other 3 bytes pack a 12-bit x and a 12-bit y.
func touchFromRecord(b []byte, w, h int) Touch {
	x := int(b[1]) | int(b[2]&0x0f)<<8
	y := int(b[2]>>4) | int(b[3])<<4
	return Touch{
		Active: b[0]&0x80 == 0,
		ID:     b[0] & 0x7f,
		X:      touchCoord(x, w),
		Y:      touchCoord(y, h),
	}
}

// touchesFromPayload decodes the touchpad records of a state payload, or
// returns prev when the payload carries no touch sample: the simplified
// Bluetooth report is too short to hold one, and a DualShock 4 report with a
// packet count of 0 holds none.
func touchesFromPayload(model model, p []byte, prev [TouchCount]Touch) [TouchCount]Touch {
	var offset int
	switch model {
	case modelDualShock4:
		// Byte 32 counts the packets that follow, each a timestamp byte and
		// two records, oldest first.
		const packetSize = 1 + 4*TouchCount
		if len(p) < 33+packetSize {
			return prev
		}
		n := int(p[32])
		if room := (len(p) - 33) / packetSize; n > room {
			n = room
		}
		if n == 0 {
			return prev
		}
		// The last packet the payload has room for is the most recent
		// sample, and a finger lifted in it stays lifted when the reports
		// that follow carry no packet.
		offset = 34 + packetSize*(n-1)
	case modelDualSense:
		// The two records are at byte 32.
		if len(p) < 32+4*TouchCount {
			return prev
		}
		offset = 32
	default:
		return prev
	}
	w, h := touchpadSize(model)
	var touches [TouchCount]Touch
	for i := range touches {
		touches[i] = touchFromRecord(p[offset+4*i:offset+4*i+4], w, h)
	}
	return touches
}

// InputState is the stick, trigger, D-pad, button, and touchpad state of a
// controller.
type InputState struct {
	// LX, LY, RX, and RY are the stick positions. 0 is left or up, and 0x80
	// is the center.
	LX, LY, RX, RY byte

	// L2 and R2 are the trigger positions. 0 is released.
	L2, R2 byte

	// Hat is the D-pad direction: 0 is up, and each step of 1 turns 45
	// degrees clockwise. 8 or above is centered.
	Hat byte

	// Buttons has bit i set while button i+1 is pressed:
	//
	//	0 Square, 1 Cross, 2 Circle, 3 Triangle, 4 L1, 5 R1, 6 L2, 7 R2,
	//	8 Share (Create), 9 Options, 10 L3, 11 R3, 12 PS, 13 Touchpad,
	//	14 Mute (DualSense only).
	Buttons uint16

	// Touches are the fingers on the touchpad, by slot.
	Touches [TouchCount]Touch
}

// neutralInputState is the state of a controller at rest: sticks centered,
// triggers released, D-pad centered, and no buttons pressed. It stands in
// until the first input report is decoded.
var neutralInputState = InputState{
	LX: 0x80, LY: 0x80, RX: 0x80, RY: 0x80,
	Hat: 8,
}

// decodeButtons decodes the three button bytes shared by all the report
// layouts. b0 carries the D-pad direction in its low nibble and the face
// buttons in its high nibble; b1 carries the shoulders, triggers, Share,
// Options, and stick buttons; b2 carries PS, Touchpad, and, on the DualSense,
// Mute in its low bits. mask selects the bits of b2 that are buttons: the
// remaining bits are a counter on the DualShock 4 and reserved on the
// DualSense.
func decodeButtons(b0, b1, b2, mask byte) (hat byte, buttons uint16) {
	return b0 & 0x0f, uint16(b0>>4) | uint16(b1)<<4 | uint16(b2&mask)<<12
}

// inputStateDS4Layout decodes the state payload of a DualShock 4 input
// report. p must be at least 9 bytes. The DualSense uses the same layout in
// its simplified Bluetooth report, without a Mute bit.
func inputStateDS4Layout(p []byte) InputState {
	hat, buttons := decodeButtons(p[4], p[5], p[6], 0x03)
	return InputState{
		LX: p[0], LY: p[1], RX: p[2], RY: p[3],
		L2: p[7], R2: p[8],
		Hat:     hat,
		Buttons: buttons,
	}
}

// inputStateDualSenseLayout decodes the state payload of a DualSense full
// input report. p must be at least 10 bytes.
func inputStateDualSenseLayout(p []byte) InputState {
	// p[6] is a counter.
	hat, buttons := decodeButtons(p[7], p[8], p[9], 0x07)
	return InputState{
		LX: p[0], LY: p[1], RX: p[2], RY: p[3],
		L2: p[4], R2: p[5],
		Hat:     hat,
		Buttons: buttons,
	}
}

// inputStateFromReport decodes an input report received from a controller
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
//
// prev is the state decoded before; its touches carry over when the report
// has no touch sample (see touchesFromPayload).
func inputStateFromReport(model model, bt bool, report []byte, prev InputState) (state InputState, ok bool) {
	if len(report) == 0 {
		return InputState{}, false
	}
	var offset, size int
	var decode func([]byte) InputState
	var crc bool
	switch {
	case report[0] == 0x01 && !bt && model == modelDualShock4:
		offset, size, decode = 1, dualShock4InputReportSizeUSB, inputStateDS4Layout
	case report[0] == 0x01 && !bt && model == modelDualSense:
		offset, size, decode = 1, dualSenseInputReportSizeUSB, inputStateDualSenseLayout
	case report[0] == 0x01 && bt && (model == modelDualShock4 || model == modelDualSense):
		offset, size, decode = 1, simpleInputReportSizeBT, inputStateDS4Layout
	case report[0] == 0x11 && bt && model == modelDualShock4:
		offset, size, decode, crc = 3, dualShock4InputReportSizeBT, inputStateDS4Layout, true
	case report[0] == 0x31 && bt && model == modelDualSense:
		offset, size, decode, crc = 2, dualSenseInputReportSizeBT, inputStateDualSenseLayout, true
	default:
		return InputState{}, false
	}
	if len(report) < size {
		return InputState{}, false
	}
	report = report[:size]
	if crc && !btInputCRCValid(report) {
		return InputState{}, false
	}
	if report[0] == 0x11 && report[1]&0x80 == 0 {
		return InputState{}, false
	}
	state = decode(report[offset:])
	state.Touches = touchesFromPayload(model, report[offset:], prev.Touches)
	return state, true
}

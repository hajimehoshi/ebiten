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

package sonyhid_test

import (
	"encoding/binary"
	"hash/crc32"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad/sonyhid"
)

func TestModelFromIDs(t *testing.T) {
	tests := []struct {
		vid  uint16
		pid  uint16
		want sonyhid.Model
	}{
		{
			vid:  0x054c,
			pid:  0x05c4,
			want: sonyhid.ModelDualShock4,
		},
		{
			vid:  0x054c,
			pid:  0x09cc,
			want: sonyhid.ModelDualShock4,
		},
		{
			vid:  0x054c,
			pid:  0x0ce6,
			want: sonyhid.ModelDualSense,
		},
		{
			vid:  0x054c,
			pid:  0x0df2,
			want: sonyhid.ModelDualSense,
		},
		// DualShock 3
		{
			vid:  0x054c,
			pid:  0x0268,
			want: sonyhid.ModelNone,
		},
		{
			vid:  0x054c,
			pid:  0x0000,
			want: sonyhid.ModelNone,
		},
		// Sony PID with a non-Sony VID
		{
			vid:  0x045e,
			pid:  0x05c4,
			want: sonyhid.ModelNone,
		},
		{
			vid:  0x0000,
			pid:  0x0000,
			want: sonyhid.ModelNone,
		},
	}
	for _, tt := range tests {
		if got := sonyhid.ModelFromIDs(tt.vid, tt.pid); got != tt.want {
			t.Errorf("ModelFromIDs(%#04x, %#04x) = %d, want %d", tt.vid, tt.pid, got, tt.want)
		}
	}
}

func TestBluetoothFromDeviceInstanceID(t *testing.T) {
	tests := []struct {
		id     string
		wantBT bool
		wantOK bool
	}{
		{
			id:     `BTHENUM\{00001124-0000-1000-8000-00805F9B34FB}_VID&0002054C_PID&09CC\8&1234ABCD&0&F0F0F0F0F0F0_C0DE0000`,
			wantBT: true,
			wantOK: true,
		},
		{
			id:     `bthenum\{00001124-0000-1000-8000-00805f9b34fb}_vid&0002054c_pid&0ce6\8&1234abcd&0&f0f0f0f0f0f0_c0de0000`,
			wantBT: true,
			wantOK: true,
		},
		{
			id:     `USB\VID_054C&PID_09CC&MI_03\9&ABCD1234&0&0003`,
			wantBT: false,
			wantOK: true,
		},
		{
			id:     `usb\vid_054c&pid_0ce6\1234567890`,
			wantBT: false,
			wantOK: true,
		},
		{
			id:     `HID\VID_054C&PID_09CC\7&1234ABCD&0&0000`,
			wantBT: false,
			wantOK: false,
		},
		{
			id:     `BTHLEDevice\{00001812-0000-1000-8000-00805F9B34FB}_Dev_VID&02054C\9&1234`,
			wantBT: false,
			wantOK: false,
		},
		// No separator; not a valid enumerator prefix.
		{
			id:     `BTHENUM`,
			wantBT: false,
			wantOK: false,
		},
		{
			id:     `USBSTOR\Disk&Ven_X&Prod_Y\5&1234`,
			wantBT: false,
			wantOK: false,
		},
		{
			id:     ``,
			wantBT: false,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		bt, ok := sonyhid.BluetoothFromDeviceInstanceID(tt.id)
		if bt != tt.wantBT || ok != tt.wantOK {
			t.Errorf("BluetoothFromDeviceInstanceID(%q) = %t, %t, want %t, %t", tt.id, bt, ok, tt.wantBT, tt.wantOK)
		}
	}
}

func TestOutputReportSize(t *testing.T) {
	tests := []struct {
		model sonyhid.Model
		bt    bool
		want  int
	}{
		{
			model: sonyhid.ModelDualShock4,
			bt:    false,
			want:  32,
		},
		{
			model: sonyhid.ModelDualShock4,
			bt:    true,
			want:  78,
		},
		{
			model: sonyhid.ModelDualSense,
			bt:    false,
			want:  48,
		},
		{
			model: sonyhid.ModelDualSense,
			bt:    true,
			want:  78,
		},
		{
			model: sonyhid.ModelNone,
			bt:    false,
			want:  0,
		},
		{
			model: sonyhid.ModelNone,
			bt:    true,
			want:  0,
		},
	}
	for _, tt := range tests {
		if got := sonyhid.OutputReportSize(tt.model, tt.bt); got != tt.want {
			t.Errorf("OutputReportSize(%d, %t) = %d, want %d", tt.model, tt.bt, got, tt.want)
		}
	}
}

func TestRumbleByte(t *testing.T) {
	tests := []struct {
		in   float64
		want byte
	}{
		{
			in:   0,
			want: 0,
		},
		{
			in:   -1,
			want: 0,
		},
		{
			in:   1,
			want: 0xff,
		},
		{
			in:   2,
			want: 0xff,
		},
		{
			in:   0.5,
			want: 0x7f,
		},
		{
			in:   math.NaN(),
			want: 0,
		},
		{
			in:   math.Inf(1),
			want: 0xff,
		},
		{
			in:   math.Inf(-1),
			want: 0,
		},
	}
	for _, tt := range tests {
		if got := sonyhid.RumbleByte(tt.in); got != tt.want {
			t.Errorf("RumbleByte(%v) = %#02x, want %#02x", tt.in, got, tt.want)
		}
	}
}

func TestBTCRC(t *testing.T) {
	for _, header := range []byte{0xa1, 0xa2} {
		if got, want := sonyhid.BTCRC(header, nil), crc32.ChecksumIEEE([]byte{header}); got != want {
			t.Errorf("BTCRC(%#02x, nil) = %#08x, want %#08x", header, got, want)
		}
		data := []byte{0x11, 0xc0, 0x00, 0x01}
		if got, want := sonyhid.BTCRC(header, data), crc32.ChecksumIEEE(append([]byte{header}, data...)); got != want {
			t.Errorf("BTCRC(%#02x, %v) = %#08x, want %#08x", header, data, got, want)
		}
	}
	if sonyhid.BTCRC(0xa1, nil) == sonyhid.BTCRC(0xa2, nil) {
		t.Errorf("BTCRC: input and output headers produce the same CRC")
	}
}

// checkReport verifies the length and the expected non-zero bytes of a
// report. want maps an offset to its expected value; every other byte must be
// 0, except the trailing 4 CRC bytes when hasCRC is set, which must match
// BTCRC with the output header over the rest of the report.
func checkReport(t *testing.T, name string, report []byte, size int, want map[int]byte, hasCRC bool) {
	t.Helper()

	if len(report) != size {
		t.Errorf("%s: len = %d, want %d", name, len(report), size)
		return
	}

	body := report
	if hasCRC {
		n := len(report) - 4
		body = report[:n]
		if got, wantCRC := binary.LittleEndian.Uint32(report[n:]), sonyhid.BTCRC(0xa2, body); got != wantCRC {
			t.Errorf("%s: CRC = %#08x, want %#08x", name, got, wantCRC)
		}
	}

	for i, b := range body {
		if w := want[i]; b != w {
			t.Errorf("%s: byte %d = %#02x, want %#02x", name, i, b, w)
		}
	}
}

func TestDualShock4RumbleReportUSB(t *testing.T) {
	checkReport(t, "usb", sonyhid.DualShock4RumbleReportUSB(0xab, 0xcd), sonyhid.DualShock4OutputReportSizeUSB, map[int]byte{
		0: 0x05,
		1: 0x01,
		4: 0xcd, // weak
		5: 0xab, // strong
	}, false)
	checkReport(t, "usb stop", sonyhid.DualShock4RumbleReportUSB(0, 0), sonyhid.DualShock4OutputReportSizeUSB, map[int]byte{
		0: 0x05,
		1: 0x01,
	}, false)
}

func TestDualShock4RumbleReportBT(t *testing.T) {
	checkReport(t, "bt", sonyhid.DualShock4RumbleReportBT(0xab, 0xcd), sonyhid.DualShock4OutputReportSizeBT, map[int]byte{
		0: 0x11,
		1: 0xc0,
		3: 0x01,
		6: 0xcd, // weak
		7: 0xab, // strong
	}, true)
}

func TestDualSenseRumbleReportUSB(t *testing.T) {
	checkReport(t, "usb", sonyhid.DualSenseRumbleReportUSB(0xab, 0xcd), sonyhid.DualSenseOutputReportSizeUSB, map[int]byte{
		0: 0x02,
		1: 0x03,
		3: 0xcd, // weak
		4: 0xab, // strong
	}, false)
}

func TestDualSenseRumbleReportBT(t *testing.T) {
	checkReport(t, "bt", sonyhid.DualSenseRumbleReportBT(2, 0xab, 0xcd), sonyhid.DualSenseOutputReportSizeBT, map[int]byte{
		0: 0x31,
		1: 0x20, // Sequence 2 in the high nibble.
		2: 0x10,
		3: 0x03,
		5: 0xcd, // weak
		6: 0xab, // strong
	}, true)

	// Only the low 4 bits of the sequence counter are used.
	r := sonyhid.DualSenseRumbleReportBT(0x1f, 0, 0)
	if got, want := r[1], byte(0xf0); got != want {
		t.Errorf("bt seq: byte 1 = %#02x, want %#02x", got, want)
	}
}

func TestInputReportSize(t *testing.T) {
	tests := []struct {
		model sonyhid.Model
		bt    bool
		want  int
	}{
		{model: sonyhid.ModelDualShock4, bt: false, want: 64},
		{model: sonyhid.ModelDualShock4, bt: true, want: 78},
		{model: sonyhid.ModelDualSense, bt: false, want: 64},
		{model: sonyhid.ModelDualSense, bt: true, want: 78},
		{model: sonyhid.ModelNone, bt: false, want: 0},
		{model: sonyhid.ModelNone, bt: true, want: 0},
	}
	for _, tt := range tests {
		if got := sonyhid.InputReportSize(tt.model, tt.bt); got != tt.want {
			t.Errorf("InputReportSize(%d, %t) = %d, want %d", tt.model, tt.bt, got, tt.want)
		}
	}
}

// ds4LayoutPayload is a state payload in the DualShock 4 layout with every
// field set to a distinct value. The third button byte carries the counter in
// its high 6 bits and the Mute bit of the DualSense layout, none of which are
// buttons in this layout.
var ds4LayoutPayload = []byte{
	0x10, 0x20, 0x30, 0x40, // lx, ly, rx, ry
	0x18,       // hat 8 (centered), Square
	0x21,       // L1, Options
	0xfe,       // Touchpad, Mute position, counter
	0x50, 0x60, // l2, r2
}

var ds4LayoutState = sonyhid.InputState{
	LX: 0x10, LY: 0x20, RX: 0x30, RY: 0x40,
	L2: 0x50, R2: 0x60,
	Hat:     8,
	Buttons: 1<<0 | 1<<4 | 1<<9 | 1<<13,
}

// dualSenseLayoutPayload is a state payload in the DualSense full layout with
// every field set to a distinct value.
var dualSenseLayoutPayload = []byte{
	0x10, 0x20, 0x30, 0x40, // lx, ly, rx, ry
	0x50, 0x60, // l2, r2
	0x99, // counter
	0x83, // hat 3 (right-down), Triangle
	0xc8, // R2, L3, R3
	0xfd, // PS, Mute, reserved
}

var dualSenseLayoutState = sonyhid.InputState{
	LX: 0x10, LY: 0x20, RX: 0x30, RY: 0x40,
	L2: 0x50, R2: 0x60,
	Hat:     3,
	Buttons: 1<<3 | 1<<7 | 1<<10 | 1<<11 | 1<<12 | 1<<14,
}

// inputReport builds an input report of the given size with the payload at
// the given offset. The other bytes are filler that must not affect decoding.
func inputReport(id byte, size, offset int, payload []byte) []byte {
	r := make([]byte, size)
	for i := range r {
		r[i] = 0xee
	}
	r[0] = id
	copy(r[offset:], payload)
	return r
}

// fullBTInputReport builds a full Bluetooth input report like inputReport,
// with the HID-data-present flag set in byte 1 and a valid CRC in the last 4
// bytes. The CRC is computed here, independently of the package, over the
// input header byte and the report bytes before the CRC.
func fullBTInputReport(id byte, size, offset int, payload []byte) []byte {
	r := inputReport(id, size, offset, payload)
	r[1] = 0xc0
	return signBTInputReport(r)
}

// signBTInputReport overwrites the last 4 bytes of a full Bluetooth input
// report with the CRC over the bytes before them.
func signBTInputReport(r []byte) []byte {
	n := len(r) - 4
	binary.LittleEndian.PutUint32(r[n:], crc32.ChecksumIEEE(append([]byte{0xa1}, r[:n]...)))
	return r
}

// corruptByte returns a copy of r with byte i inverted.
func corruptByte(r []byte, i int) []byte {
	c := append([]byte{}, r...)
	c[i] ^= 0xff
	return c
}

func TestInputStateFromReport(t *testing.T) {
	ds4USB := inputReport(0x01, sonyhid.DualShock4InputReportSizeUSB, 1, ds4LayoutPayload)
	dualSenseUSB := inputReport(0x01, sonyhid.DualSenseInputReportSizeUSB, 1, dualSenseLayoutPayload)
	simple := inputReport(0x01, sonyhid.SimpleInputReportSizeBT, 1, ds4LayoutPayload)
	ds4Full := fullBTInputReport(0x11, sonyhid.DualShock4InputReportSizeBT, 3, ds4LayoutPayload)
	dualSenseFull := fullBTInputReport(0x31, sonyhid.DualSenseInputReportSizeBT, 2, dualSenseLayoutPayload)

	// ds4FullNoData has a valid CRC but its HID-data-present flag clear, so
	// it carries no controller state.
	ds4FullNoData := append([]byte{}, ds4Full...)
	ds4FullNoData[1] = 0x40
	signBTInputReport(ds4FullNoData)

	tests := []struct {
		name   string
		model  sonyhid.Model
		bt     bool
		report []byte
		want   sonyhid.InputState
		wantOK bool
	}{
		{
			name:   "ds4 usb",
			model:  sonyhid.ModelDualShock4,
			report: ds4USB,
			want:   ds4LayoutState,
			wantOK: true,
		},
		{
			name:   "DualSense usb",
			model:  sonyhid.ModelDualSense,
			report: dualSenseUSB,
			want:   dualSenseLayoutState,
			wantOK: true,
		},
		{
			name:   "ds4 simple",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: simple,
			want:   ds4LayoutState,
			wantOK: true,
		},
		{
			name:   "ds4 full",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: ds4Full,
			want:   ds4LayoutState,
			wantOK: true,
		},
		{
			name:   "DualSense simple",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: simple,
			want:   ds4LayoutState,
			wantOK: true,
		},
		{
			name:   "DualSense full",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: dualSenseFull,
			want:   dualSenseLayoutState,
			wantOK: true,
		},
		// The host may pad reports to the device's maximum report length.
		{
			name:   "ds4 full padded",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: append(ds4Full, make([]byte, 400)...),
			want:   ds4LayoutState,
			wantOK: true,
		},
		{
			name:   "DualSense full padded",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: append(dualSenseFull, make([]byte, 400)...),
			want:   dualSenseLayoutState,
			wantOK: true,
		},
		{
			name:   "simple padded",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: append(simple, make([]byte, 60)...),
			want:   ds4LayoutState,
			wantOK: true,
		},
		// Full Bluetooth reports carry a CRC over the bytes before it. A
		// report whose CRC does not match its contents is corrupted, and is
		// rejected whether the damage is in the state or in the CRC itself.
		{
			name:   "ds4 full corrupted state",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: corruptByte(ds4Full, 3+5), // Second button byte.
		},
		{
			name:   "ds4 full corrupted crc",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: corruptByte(ds4Full, sonyhid.DualShock4InputReportSizeBT-1),
		},
		{
			name:   "DualSense full corrupted state",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: corruptByte(dualSenseFull, 2+8), // Second button byte.
		},
		{
			name:   "DualSense full corrupted crc",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: corruptByte(dualSenseFull, sonyhid.DualSenseInputReportSizeBT-1),
		},
		// The CRC covers the report, not the host's padding.
		{
			name:   "ds4 full padded corrupted",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: append(corruptByte(ds4Full, 3), make([]byte, 400)...),
		},
		{
			name:   "ds4 full no crc",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: inputReport(0x11, sonyhid.DualShock4InputReportSizeBT, 3, ds4LayoutPayload),
		},
		{
			name:   "DualSense full no crc",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: inputReport(0x31, sonyhid.DualSenseInputReportSizeBT, 2, dualSenseLayoutPayload),
		},
		// A DualShock 4 full report carries controller state only when its
		// HID-data-present flag is set.
		{
			name:   "ds4 full no hid data",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: ds4FullNoData,
		},
		// The DualSense uses different layouts for report 0x01 over USB and
		// over Bluetooth; the transport selects the layout.
		{
			name:   "DualSense usb report over bt",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: dualSenseUSB,
			want: sonyhid.InputState{
				LX: 0x10, LY: 0x20, RX: 0x30, RY: 0x40,
				L2: 0x83, R2: 0xc8,
				Hat:     0,
				Buttons: 0x5<<0 | 0x60<<4 | 0x1<<12, // 0x50 >> 4, 0x60, 0x99 & 0x03
			},
			wantOK: true,
		},
		// Reports of the other model, of the other transport, or of other
		// kinds are not state reports.
		{
			name:   "ds4 gets DualSense full",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: dualSenseFull,
		},
		{
			name:   "DualSense gets ds4 full",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: ds4Full,
		},
		{
			name:   "ds4 full over usb",
			model:  sonyhid.ModelDualShock4,
			report: ds4Full,
		},
		{
			name:   "DualSense full over usb",
			model:  sonyhid.ModelDualSense,
			report: dualSenseFull,
		},
		{
			name:   "simple over usb",
			model:  sonyhid.ModelDualShock4,
			report: simple,
		},
		{
			name:   "unknown model usb",
			model:  sonyhid.ModelNone,
			report: ds4USB,
		},
		{
			name:   "unknown model bt",
			model:  sonyhid.ModelNone,
			bt:     true,
			report: simple,
		},
		{
			name:   "other report id",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: inputReport(0x12, sonyhid.DualShock4InputReportSizeBT, 3, ds4LayoutPayload),
		},
		{
			name:   "empty",
			model:  sonyhid.ModelDualShock4,
			report: nil,
		},
		{
			name:   "ds4 usb short",
			model:  sonyhid.ModelDualShock4,
			report: ds4USB[:sonyhid.DualShock4InputReportSizeUSB-1],
		},
		{
			name:   "DualSense usb short",
			model:  sonyhid.ModelDualSense,
			report: dualSenseUSB[:sonyhid.DualSenseInputReportSizeUSB-1],
		},
		{
			name:   "simple short",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: simple[:sonyhid.SimpleInputReportSizeBT-1],
		},
		{
			name:   "ds4 full short",
			model:  sonyhid.ModelDualShock4,
			bt:     true,
			report: ds4Full[:sonyhid.DualShock4InputReportSizeBT-1],
		},
		{
			name:   "DualSense full short",
			model:  sonyhid.ModelDualSense,
			bt:     true,
			report: dualSenseFull[:sonyhid.DualSenseInputReportSizeBT-1],
		},
	}
	for _, tt := range tests {
		got, ok := sonyhid.InputStateFromReport(tt.model, tt.bt, tt.report)
		if ok != tt.wantOK {
			t.Errorf("%s: ok = %t, want %t", tt.name, ok, tt.wantOK)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: state = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestInputStateFromReportHat(t *testing.T) {
	for hat := range byte(16) {
		payload := append([]byte{}, ds4LayoutPayload...)
		payload[4] = hat | 0xf0
		got, ok := sonyhid.InputStateFromReport(sonyhid.ModelDualShock4, false, inputReport(0x01, sonyhid.DualShock4InputReportSizeUSB, 1, payload))
		if !ok {
			t.Fatalf("hat %d: not ok", hat)
		}
		if got.Hat != hat {
			t.Errorf("hat %d: got %d", hat, got.Hat)
		}
		if want := uint16(0x0f | 1<<4 | 1<<9 | 1<<13); got.Buttons != want {
			t.Errorf("hat %d: buttons = %#04x, want %#04x", hat, got.Buttons, want)
		}
	}
}

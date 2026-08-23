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
	"math"
	"testing"
)

func TestSonyModelFromIDs(t *testing.T) {
	tests := []struct {
		vid, pid uint16
		want     sonyModel
	}{
		{0x054c, 0x05c4, sonyModelDualShock4},
		{0x054c, 0x09cc, sonyModelDualShock4},
		{0x054c, 0x0ce6, sonyModelDualSense},
		{0x054c, 0x0df2, sonyModelDualSense},
		{0x054c, 0x0268, sonyModelNone}, // DualShock 3
		{0x054c, 0x0000, sonyModelNone},
		{0x045e, 0x05c4, sonyModelNone}, // Sony PID with a non-Sony VID
		{0x0000, 0x0000, sonyModelNone},
	}
	for _, tt := range tests {
		if got := sonyModelFromIDs(tt.vid, tt.pid); got != tt.want {
			t.Errorf("sonyModelFromIDs(%#04x, %#04x) = %d, want %d", tt.vid, tt.pid, got, tt.want)
		}
	}
}

func TestBluetoothFromDeviceInstanceID(t *testing.T) {
	tests := []struct {
		id     string
		wantBT bool
		wantOK bool
	}{
		{`BTHENUM\{00001124-0000-1000-8000-00805F9B34FB}_VID&0002054C_PID&09CC\8&1234ABCD&0&F0F0F0F0F0F0_C0DE0000`, true, true},
		{`bthenum\{00001124-0000-1000-8000-00805f9b34fb}_vid&0002054c_pid&0ce6\8&1234abcd&0&f0f0f0f0f0f0_c0de0000`, true, true},
		{`USB\VID_054C&PID_09CC&MI_03\9&ABCD1234&0&0003`, false, true},
		{`usb\vid_054c&pid_0ce6\1234567890`, false, true},
		{`HID\VID_054C&PID_09CC\7&1234ABCD&0&0000`, false, false},
		{`BTHLEDevice\{00001812-0000-1000-8000-00805F9B34FB}_Dev_VID&02054C\9&1234`, false, false},
		{`BTHENUM`, false, false}, // No separator; not a valid enumerator prefix.
		{`USBSTOR\Disk&Ven_X&Prod_Y\5&1234`, false, false},
		{``, false, false},
	}
	for _, tt := range tests {
		bt, ok := bluetoothFromDeviceInstanceID(tt.id)
		if bt != tt.wantBT || ok != tt.wantOK {
			t.Errorf("bluetoothFromDeviceInstanceID(%q) = %t, %t, want %t, %t", tt.id, bt, ok, tt.wantBT, tt.wantOK)
		}
	}
}

func TestSonyReportSize(t *testing.T) {
	tests := []struct {
		model sonyModel
		bt    bool
		want  int
	}{
		{sonyModelDualShock4, false, 32},
		{sonyModelDualShock4, true, 78},
		{sonyModelDualSense, false, 48},
		{sonyModelDualSense, true, 78},
		{sonyModelNone, false, 0},
		{sonyModelNone, true, 0},
	}
	for _, tt := range tests {
		if got := sonyReportSize(tt.model, tt.bt); got != tt.want {
			t.Errorf("sonyReportSize(%d, %t) = %d, want %d", tt.model, tt.bt, got, tt.want)
		}
	}
}

func TestSonyRumbleByte(t *testing.T) {
	tests := []struct {
		in   float64
		want byte
	}{
		{0, 0},
		{-1, 0},
		{1, 0xff},
		{2, 0xff},
		{0.5, 0x7f},
		{math.NaN(), 0},
		{math.Inf(1), 0xff},
		{math.Inf(-1), 0},
	}
	for _, tt := range tests {
		if got := sonyRumbleByte(tt.in); got != tt.want {
			t.Errorf("sonyRumbleByte(%v) = %#02x, want %#02x", tt.in, got, tt.want)
		}
	}
}

func TestSonyBTCRC(t *testing.T) {
	if got, want := sonyBTCRC(nil), crc32.ChecksumIEEE([]byte{0xa2}); got != want {
		t.Errorf("sonyBTCRC(nil) = %#08x, want %#08x", got, want)
	}
	data := []byte{0x11, 0xc0, 0x00, 0x01}
	if got, want := sonyBTCRC(data), crc32.ChecksumIEEE(append([]byte{0xa2}, data...)); got != want {
		t.Errorf("sonyBTCRC(%v) = %#08x, want %#08x", data, got, want)
	}
}

// checkReport verifies the length and the expected non-zero bytes of a
// report. want maps an offset to its expected value; every other byte must be
// 0, except the trailing 4 CRC bytes when hasCRC is set, which must match
// sonyBTCRC over the rest of the report.
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
		if got, wantCRC := binary.LittleEndian.Uint32(report[n:]), sonyBTCRC(body); got != wantCRC {
			t.Errorf("%s: CRC = %#08x, want %#08x", name, got, wantCRC)
		}
	}

	for i, b := range body {
		if w := want[i]; b != w {
			t.Errorf("%s: byte %d = %#02x, want %#02x", name, i, b, w)
		}
	}
}

func TestDualshock4RumbleReportUSB(t *testing.T) {
	checkReport(t, "usb", dualshock4RumbleReportUSB(0xab, 0xcd), dualshock4ReportSizeUSB, map[int]byte{
		0: 0x05,
		1: 0x01,
		4: 0xcd, // weak
		5: 0xab, // strong
	}, false)
	checkReport(t, "usb stop", dualshock4RumbleReportUSB(0, 0), dualshock4ReportSizeUSB, map[int]byte{
		0: 0x05,
		1: 0x01,
	}, false)
}

func TestDualshock4RumbleReportBT(t *testing.T) {
	checkReport(t, "bt", dualshock4RumbleReportBT(0xab, 0xcd), dualshock4ReportSizeBT, map[int]byte{
		0: 0x11,
		1: 0xc0,
		3: 0x01,
		6: 0xcd, // weak
		7: 0xab, // strong
	}, true)
}

func TestDualsenseRumbleReportUSB(t *testing.T) {
	checkReport(t, "usb", dualsenseRumbleReportUSB(0xab, 0xcd), dualsenseReportSizeUSB, map[int]byte{
		0: 0x02,
		1: 0x03,
		3: 0xcd, // weak
		4: 0xab, // strong
	}, false)
}

func TestDualsenseRumbleReportBT(t *testing.T) {
	checkReport(t, "bt", dualsenseRumbleReportBT(2, 0xab, 0xcd), dualsenseReportSizeBT, map[int]byte{
		0: 0x31,
		1: 0x20, // Sequence 2 in the high nibble.
		2: 0x10,
		3: 0x03,
		5: 0xcd, // weak
		6: 0xab, // strong
	}, true)

	// Only the low 4 bits of the sequence counter are used.
	r := dualsenseRumbleReportBT(0x1f, 0, 0)
	if got, want := r[1], byte(0xf0); got != want {
		t.Errorf("bt seq: byte 1 = %#02x, want %#02x", got, want)
	}
}

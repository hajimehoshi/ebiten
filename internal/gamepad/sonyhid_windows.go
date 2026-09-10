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
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

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

// sonyDevice is the HID device of a PlayStation controller. It drives rumble
// by writing output reports to the device, and supplies the controller's
// input by reading its input reports, in place of the DirectInput device the
// controller was enumerated through.
//
// Input cannot be left to DirectInput: over Bluetooth, the first output report
// switches the controller from its simplified input report to its full one
// for the rest of the session, and DirectInput only sees the simplified
// report, so its state freezes from then on. USB input is read the same way
// so that both transports share one path, which the full reports also need
// for input DirectInput cannot expose. See sonyInputStateFromReport.
//
// All I/O is overlapped and never waited on: the methods are called with the
// gamepad mutex held, and a synchronous operation on a stalled device (e.g. a
// dying Bluetooth link) would block gamepad polling. At most one write and one
// read are in flight. A write requested while one is pending cancels the
// pending write, and the requested motor state is held as the last event and
// issued from update once the canceled write concludes and frees the device.
type sonyDevice struct {
	handle windows.Handle
	model  sonyModel
	bt     bool
	seq    byte

	// wbuf is the reused WriteFile buffer. Its length is the device's maximum
	// output report length, which WriteFile requires regardless of the actual
	// report's length; the padding beyond the report stays zero. While a
	// write is pending the kernel owns the buffer, and it must not be
	// modified until the write completes.
	wbuf     []byte
	wov      windows.Overlapped
	wpending bool

	// nextStrong and nextWeak hold the last motor state requested while a
	// write was pending. next reports whether they are waiting to be issued.
	next       bool
	nextStrong byte
	nextWeak   byte

	vib    bool
	vibEnd time.Time

	// rbuf is the reused ReadFile buffer. Its length is the device's maximum
	// input report length, which ReadFile requires. The kernel owns it while
	// a read is pending.
	rbuf     []byte
	rov      windows.Overlapped
	rpending bool

	// input is the state decoded from the newest input report. inputLost is
	// set once a read has failed, which happens when the controller
	// disconnects; no further reads are issued.
	input     sonyInputState
	inputLost bool
}

// maxInputReportsPerUpdate bounds the reports consumed by one updateInput
// call, so that a device flooding reports cannot stall gamepad polling. A
// controller sends at most a few reports per millisecond.
const maxInputReportsPerUpdate = 64

// parentDeviceInstanceID resolves a device interface path to its device node
// and returns the device instance ID of the node's parent. A device instance
// ID begins with the name of the enumerator that created the device, so the
// parent's ID names the bus the device is connected through.
// See https://learn.microsoft.com/en-us/windows-hardware/drivers/install/device-instance-ids.
func parentDeviceInstanceID(interfacePath string) (string, error) {
	for _, p := range []*windows.LazyProc{procCMGetDeviceInterfacePropertyW, procCMLocateDevNodeW, procCMGetParent, procCMGetDeviceIDW} {
		if err := p.Find(); err != nil {
			return "", err
		}
	}

	pathPtr, err := windows.UTF16PtrFromString(interfacePath)
	if err != nil {
		return "", err
	}

	var propType windows.DEVPROPTYPE
	var size uint32
	if r, _, _ := procCMGetDeviceInterfacePropertyW.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&devpkeyDeviceInstanceID)),
		uintptr(unsafe.Pointer(&propType)), 0, uintptr(unsafe.Pointer(&size)), 0); windows.CONFIGRET(r) != windows.CR_BUFFER_SMALL {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if size == 0 || size%2 != 0 {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW returned an invalid size: %d", size)
	}

	instanceID := make([]uint16, size/2)
	if r, _, _ := procCMGetDeviceInterfacePropertyW.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&devpkeyDeviceInstanceID)),
		uintptr(unsafe.Pointer(&propType)), uintptr(unsafe.Pointer(&instanceID[0])), uintptr(unsafe.Pointer(&size)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if propType != windows.DEVPROP_TYPE_STRING {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW returned an unexpected property type: %d", propType)
	}

	var devInst windows.DEVINST
	if r, _, _ := procCMLocateDevNodeW.Call(uintptr(unsafe.Pointer(&devInst)), uintptr(unsafe.Pointer(&instanceID[0])),
		_CM_LOCATE_DEVNODE_NORMAL); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Locate_DevNodeW failed: CONFIGRET(%d)", r)
	}

	var parent windows.DEVINST
	if r, _, _ := procCMGetParent.Call(uintptr(unsafe.Pointer(&parent)), uintptr(devInst), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Parent failed: CONFIGRET(%d)", r)
	}

	var parentID [windows.MAX_DEVICE_ID_LEN + 1]uint16
	if r, _, _ := procCMGetDeviceIDW.Call(uintptr(parent), uintptr(unsafe.Pointer(&parentID[0])),
		uintptr(len(parentID)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Device_IDW failed: CONFIGRET(%d)", r)
	}

	return windows.UTF16ToString(parentID[:]), nil
}

// hidCaps returns the capabilities of an opened HID device. The report byte
// lengths in the capabilities are maximums and include the report ID byte.
func hidCaps(handle windows.Handle) (_HIDP_CAPS, error) {
	for _, p := range []*windows.LazyProc{procHidDGetPreparsedData, procHidDFreePreparsedData, procHidPGetCaps} {
		if err := p.Find(); err != nil {
			return _HIDP_CAPS{}, err
		}
	}

	var preparsedData uintptr
	if r, _, _ := procHidDGetPreparsedData.Call(uintptr(handle), uintptr(unsafe.Pointer(&preparsedData))); r == 0 {
		return _HIDP_CAPS{}, fmt.Errorf("gamepad: HidD_GetPreparsedData failed")
	}
	defer func() {
		_, _, _ = procHidDFreePreparsedData.Call(preparsedData)
	}()

	var caps _HIDP_CAPS
	if r, _, _ := procHidPGetCaps.Call(preparsedData, uintptr(unsafe.Pointer(&caps))); uint32(r) != _HIDP_STATUS_SUCCESS {
		return _HIDP_CAPS{}, fmt.Errorf("gamepad: HidP_GetCaps failed: NTSTATUS(%#08x)", uint32(r))
	}

	return caps, nil
}

// openSonyDevice opens the HID device backing a PlayStation controller for
// writing rumble output reports and reading input reports. It returns nil if
// the IDs do not identify a known PlayStation controller, or if the device
// cannot be opened and probed, e.g. when another application holds it
// exclusively. A nil result means the gamepad works without rumble, with its
// input read through DirectInput.
func openSonyDevice(path string, vid, pid uint16) *sonyDevice {
	model := sonyModelFromIDs(vid, pid)
	if model == sonyModelNone {
		return nil
	}

	// The device is reached through the same HID output reports over USB and
	// Bluetooth, but the report framing differs, so the transport must be
	// known. The enumerator of the HID device node's parent names it.
	parentID, err := parentDeviceInstanceID(path)
	if err != nil {
		return nil
	}
	bt, ok := bluetoothFromDeviceInstanceID(parentID)
	if !ok {
		return nil
	}

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	handle, err := windows.CreateFile(pathPtr, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil
	}

	caps, err := hidCaps(handle)
	if err != nil || int(caps.OutputReportByteLength) < sonyOutputReportSize(model, bt) ||
		int(caps.InputReportByteLength) < sonyInputReportSize(model, bt) {
		_ = windows.CloseHandle(handle)
		return nil
	}

	s := &sonyDevice{
		handle: handle,
		model:  model,
		bt:     bt,
		wbuf:   make([]byte, caps.OutputReportByteLength),
		rbuf:   make([]byte, caps.InputReportByteLength),
		input:  sonyNeutralInputState,
	}
	if s.wov.HEvent, err = windows.CreateEvent(nil, 1, 0, nil); err != nil {
		s.close()
		return nil
	}
	if s.rov.HEvent, err = windows.CreateEvent(nil, 1, 0, nil); err != nil {
		s.close()
		return nil
	}
	return s
}

func (s *sonyDevice) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		s.stop()
		return
	}
	s.vib = true
	s.vibEnd = time.Now().Add(duration)
	s.write(sonyRumbleByte(strongMagnitude), sonyRumbleByte(weakMagnitude))
}

func (s *sonyDevice) stop() {
	s.vib = false
	s.write(0, 0)
}

// update issues any motor state held behind a pending write, and stops the
// rumble once the requested duration has passed, as the controller keeps
// rumbling until told otherwise.
func (s *sonyDevice) update() {
	if s.wpending && s.pollWrite() && s.next {
		s.next = false
		s.issueWrite(s.nextStrong, s.nextWeak)
	}
	if s.vib && time.Since(s.vibEnd) >= 0 {
		s.stop()
	}
}

func (s *sonyDevice) write(strong, weak byte) {
	if s.wpending && !s.pollWrite() {
		// The pending write is superseded; ask it to conclude. Cancellation
		// completes asynchronously, so the new state cannot be written until
		// the canceled write is observed done and the buffer is free again.
		_ = windows.CancelIoEx(s.handle, &s.wov)
		s.next = true
		s.nextStrong = strong
		s.nextWeak = weak
		return
	}
	// A directly issued state supersedes any held one; without this, an older
	// state could be issued after a newer one.
	s.next = false
	s.issueWrite(strong, weak)
}

// pollWrite checks the pending write without blocking and reports whether
// the device is free for another write.
func (s *sonyDevice) pollWrite() bool {
	var written uint32
	if err := windows.GetOverlappedResult(s.handle, &s.wov, &written, false); errors.Is(err, windows.ERROR_IO_INCOMPLETE) {
		return false
	}
	// The write completed; a write error means no rumble, and there is
	// nothing to report to the caller.
	s.wpending = false
	return true
}

func (s *sonyDevice) issueWrite(strong, weak byte) {
	var report []byte
	switch s.model {
	case sonyModelDualShock4:
		if s.bt {
			report = dualShock4RumbleReportBT(strong, weak)
		} else {
			report = dualShock4RumbleReportUSB(strong, weak)
		}
	case sonyModelDualSense:
		if s.bt {
			report = dualSenseRumbleReportBT(s.seq, strong, weak)
			s.seq++
		} else {
			report = dualSenseRumbleReportUSB(strong, weak)
		}
	default:
		return
	}

	copy(s.wbuf, report)
	var written uint32
	if err := windows.WriteFile(s.handle, s.wbuf, &written, &s.wov); errors.Is(err, windows.ERROR_IO_PENDING) {
		s.wpending = true
	}
	// Any other write error means no rumble; there is nothing to report to
	// the caller.
}

// updateInput consumes the input reports received since the last call and
// decodes the newest state report into s.input. It reports false once the
// device has stopped delivering input, which happens when the controller
// disconnects.
func (s *sonyDevice) updateInput() bool {
	if s.inputLost {
		return false
	}
	for range maxInputReportsPerUpdate {
		if !s.rpending {
			var n uint32
			if err := windows.ReadFile(s.handle, s.rbuf, &n, &s.rov); err != nil && !errors.Is(err, windows.ERROR_IO_PENDING) {
				s.inputLost = true
				return false
			}
			// A read that completed synchronously is collected below like a
			// pending one, which also reports its length reliably.
			s.rpending = true
		}
		var n uint32
		err := windows.GetOverlappedResult(s.handle, &s.rov, &n, false)
		if errors.Is(err, windows.ERROR_IO_INCOMPLETE) {
			return true
		}
		s.rpending = false
		if err != nil {
			s.inputLost = true
			return false
		}
		if state, ok := sonyInputStateFromReport(s.model, s.bt, s.rbuf[:min(int(n), len(s.rbuf))]); ok {
			s.input = state
		}
	}
	return true
}

// close cancels the pending I/O and releases the device. close can be called
// multiple times.
//
// The kernel writes the result of a canceled operation into its OVERLAPPED
// structure when the cancellation completes, which happens asynchronously, and
// owns the operation's buffer until then. The device is therefore kept
// reachable, and its handles open, until every pending operation has been
// observed complete. The wait happens on a separate goroutine so that
// releasing a device with a stalled link does not block gamepad polling.
func (s *sonyDevice) close() {
	if s.handle == 0 {
		return
	}
	handle := s.handle
	s.handle = 0

	// CancelIo is not usable here: it only cancels I/O issued by the calling
	// thread, and goroutines migrate between threads.
	_ = windows.CancelIoEx(handle, nil)

	wpending, rpending := s.wpending, s.rpending
	s.wpending, s.rpending = false, false
	release := func() {
		var n uint32
		if wpending {
			_ = windows.GetOverlappedResult(handle, &s.wov, &n, true)
		}
		if rpending {
			_ = windows.GetOverlappedResult(handle, &s.rov, &n, true)
		}
		_ = windows.CloseHandle(handle)
		if s.wov.HEvent != 0 {
			_ = windows.CloseHandle(s.wov.HEvent)
			s.wov.HEvent = 0
		}
		if s.rov.HEvent != 0 {
			_ = windows.CloseHandle(s.rov.HEvent)
			s.rov.HEvent = 0
		}
	}
	if !wpending && !rpending {
		release()
		return
	}
	go release()
}

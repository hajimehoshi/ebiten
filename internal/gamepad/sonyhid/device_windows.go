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
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

// Device is the HID device of a PlayStation controller.
type Device struct {
	// All I/O is overlapped and never waited on: the methods are called with the
	// gamepad mutex held, and a synchronous operation on a stalled device (e.g. a
	// dying Bluetooth link) would block gamepad polling. At most one write and one
	// read are in flight. A write requested while one is pending cancels the
	// pending write, and the requested motor state is held as the last event and
	// issued from Update once the canceled write concludes and frees the device.
	handle windows.Handle
	model  model
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
	input     InputState
	inputLost bool
}

// maxInputReportsPerUpdate bounds the reports consumed by one UpdateInput
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
		return "", fmt.Errorf("sonyhid: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if size == 0 || size%2 != 0 {
		return "", fmt.Errorf("sonyhid: CM_Get_Device_Interface_PropertyW returned an invalid size: %d", size)
	}

	instanceID := make([]uint16, size/2)
	if r, _, _ := procCMGetDeviceInterfacePropertyW.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&devpkeyDeviceInstanceID)),
		uintptr(unsafe.Pointer(&propType)), uintptr(unsafe.Pointer(&instanceID[0])), uintptr(unsafe.Pointer(&size)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("sonyhid: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if propType != windows.DEVPROP_TYPE_STRING {
		return "", fmt.Errorf("sonyhid: CM_Get_Device_Interface_PropertyW returned an unexpected property type: %d", propType)
	}

	var devInst windows.DEVINST
	if r, _, _ := procCMLocateDevNodeW.Call(uintptr(unsafe.Pointer(&devInst)), uintptr(unsafe.Pointer(&instanceID[0])),
		_CM_LOCATE_DEVNODE_NORMAL); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("sonyhid: CM_Locate_DevNodeW failed: CONFIGRET(%d)", r)
	}

	var parent windows.DEVINST
	if r, _, _ := procCMGetParent.Call(uintptr(unsafe.Pointer(&parent)), uintptr(devInst), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("sonyhid: CM_Get_Parent failed: CONFIGRET(%d)", r)
	}

	var parentID [windows.MAX_DEVICE_ID_LEN + 1]uint16
	if r, _, _ := procCMGetDeviceIDW.Call(uintptr(parent), uintptr(unsafe.Pointer(&parentID[0])),
		uintptr(len(parentID)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("sonyhid: CM_Get_Device_IDW failed: CONFIGRET(%d)", r)
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
		return _HIDP_CAPS{}, fmt.Errorf("sonyhid: HidD_GetPreparsedData failed")
	}
	defer func() {
		_, _, _ = procHidDFreePreparsedData.Call(preparsedData)
	}()

	var caps _HIDP_CAPS
	if r, _, _ := procHidPGetCaps.Call(preparsedData, uintptr(unsafe.Pointer(&caps))); uint32(r) != _HIDP_STATUS_SUCCESS {
		return _HIDP_CAPS{}, fmt.Errorf("sonyhid: HidP_GetCaps failed: NTSTATUS(%#08x)", uint32(r))
	}

	return caps, nil
}

// Open opens a PlayStation controller for input and rumble, or returns nil if
// the controller is unsupported or cannot be opened and probed.
func Open(path string, vid, pid uint16) *Device {
	model := modelFromIDs(vid, pid)
	if model == modelNone {
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
	if err != nil || int(caps.OutputReportByteLength) < outputReportSize(model, bt) ||
		int(caps.InputReportByteLength) < inputReportSize(model, bt) {
		_ = windows.CloseHandle(handle)
		return nil
	}

	s := &Device{
		handle: handle,
		model:  model,
		bt:     bt,
		wbuf:   make([]byte, caps.OutputReportByteLength),
		rbuf:   make([]byte, caps.InputReportByteLength),
		input:  neutralInputState,
	}
	if s.wov.HEvent, err = windows.CreateEvent(nil, 1, 0, nil); err != nil {
		s.Close()
		return nil
	}
	if s.rov.HEvent, err = windows.CreateEvent(nil, 1, 0, nil); err != nil {
		s.Close()
		return nil
	}
	return s
}

// Vibrate runs the motors for the duration with magnitudes clamped to 0 to 1.
func (s *Device) Vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	strongMagnitude = mathutil.Clamp01(strongMagnitude)
	weakMagnitude = mathutil.Clamp01(weakMagnitude)

	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		s.stop()
		return
	}
	s.vib = true
	s.vibEnd = time.Now().Add(duration)
	s.write(rumbleByte(strongMagnitude), rumbleByte(weakMagnitude))
}

func (s *Device) stop() {
	s.vib = false
	s.write(0, 0)
}

// Update advances rumble output and stops expired vibration.
func (s *Device) Update() {
	if s.wpending && s.pollWrite() && s.next {
		s.next = false
		s.issueWrite(s.nextStrong, s.nextWeak)
	}
	if s.vib && time.Since(s.vibEnd) >= 0 {
		s.stop()
	}
}

func (s *Device) write(strong, weak byte) {
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
func (s *Device) pollWrite() bool {
	var written uint32
	if err := windows.GetOverlappedResult(s.handle, &s.wov, &written, false); errors.Is(err, windows.ERROR_IO_INCOMPLETE) {
		return false
	}
	// The write completed; a write error means no rumble, and there is
	// nothing to report to the caller.
	s.wpending = false
	return true
}

func (s *Device) issueWrite(strong, weak byte) {
	var report []byte
	switch s.model {
	case modelDualShock4:
		if s.bt {
			report = dualShock4RumbleReportBT(strong, weak)
		} else {
			report = dualShock4RumbleReportUSB(strong, weak)
		}
	case modelDualSense:
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

// UpdateInput reads available input and reports whether the device is still connected.
func (s *Device) UpdateInput() bool {
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
		if state, ok := inputStateFromReport(s.model, s.bt, s.rbuf[:min(int(n), len(s.rbuf))]); ok {
			s.input = state
		}
	}
	return true
}

// Close releases the device.
func (s *Device) Close() {
	// The kernel writes the result of a canceled operation into its OVERLAPPED
	// structure when the cancellation completes, which happens asynchronously, and
	// owns the operation's buffer until then. The device is therefore kept
	// reachable, and its handles open, until every pending operation has been
	// observed complete. The wait happens on a separate goroutine so that
	// releasing a device with a stalled link does not block gamepad polling.

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

// Input returns the most recently read controller state.
func (s *Device) Input() InputState {
	return s.input
}

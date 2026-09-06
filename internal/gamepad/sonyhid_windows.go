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
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

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
			report = dualshock4RumbleReportBT(strong, weak)
		} else {
			report = dualshock4RumbleReportUSB(strong, weak)
		}
	case sonyModelDualSense:
		if s.bt {
			report = dualsenseRumbleReportBT(s.seq, strong, weak)
			s.seq++
		} else {
			report = dualsenseRumbleReportUSB(strong, weak)
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

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

// sonyRumbler drives rumble on a PlayStation controller by writing output
// reports to its HID device, independently of the DirectInput device used for
// reading input.
//
// Writes are overlapped with at most one in flight, and are never waited on:
// vibrate is called with the gamepad mutex held, and a synchronous write to a
// stalled device (e.g. a dying Bluetooth link) would block gamepad polling.
// A write requested while one is pending cancels the pending write, and the
// requested motor state is held as the last event and issued from update once
// the canceled write concludes and frees the device.
type sonyRumbler struct {
	handle windows.Handle
	model  sonyModel
	bt     bool
	seq    byte

	// buf is the reused WriteFile buffer. Its length is the device's maximum
	// output report length, which WriteFile requires regardless of the actual
	// report's length; the padding beyond the report stays zero. While a
	// write is pending the kernel owns the buffer, and it must not be
	// modified until the write completes.
	buf     []byte
	ov      windows.Overlapped
	pending bool

	// nextStrong and nextWeak hold the last motor state requested while a
	// write was pending. next reports whether they are waiting to be issued.
	next       bool
	nextStrong byte
	nextWeak   byte

	vib    bool
	vibEnd time.Time
}

// openSonyRumble opens the HID device backing a PlayStation controller for
// writing rumble output reports. It returns nil if the IDs do not identify a
// known PlayStation controller, or if the device cannot be opened and probed,
// e.g. when another application holds it exclusively. A nil result means the
// gamepad works without rumble.
func openSonyRumble(path string, vid, pid uint16) *sonyRumbler {
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
	handle, err := windows.CreateFile(pathPtr, windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil
	}

	outLen, err := hidOutputReportByteLength(handle)
	if err != nil || outLen < sonyReportSize(model, bt) {
		_ = windows.CloseHandle(handle)
		return nil
	}

	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(handle)
		return nil
	}

	s := &sonyRumbler{
		handle: handle,
		model:  model,
		bt:     bt,
		buf:    make([]byte, outLen),
	}
	s.ov.HEvent = event
	return s
}

func (s *sonyRumbler) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		s.stop()
		return
	}
	s.vib = true
	s.vibEnd = time.Now().Add(duration)
	s.write(sonyRumbleByte(strongMagnitude), sonyRumbleByte(weakMagnitude))
}

func (s *sonyRumbler) stop() {
	s.vib = false
	s.write(0, 0)
}

// update issues any motor state held behind a pending write, and stops the
// rumble once the requested duration has passed, as the controller keeps
// rumbling until told otherwise.
func (s *sonyRumbler) update() {
	if s.pending && s.pollWrite() && s.next {
		s.next = false
		s.issueWrite(s.nextStrong, s.nextWeak)
	}
	if s.vib && time.Now().Sub(s.vibEnd) >= 0 {
		s.stop()
	}
}

func (s *sonyRumbler) write(strong, weak byte) {
	if s.pending && !s.pollWrite() {
		// The pending write is superseded; ask it to conclude. Cancellation
		// completes asynchronously, so the new state cannot be written until
		// the canceled write is observed done and the buffer is free again.
		_ = windows.CancelIoEx(s.handle, &s.ov)
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
func (s *sonyRumbler) pollWrite() bool {
	var written uint32
	if err := windows.GetOverlappedResult(s.handle, &s.ov, &written, false); errors.Is(err, windows.ERROR_IO_INCOMPLETE) {
		return false
	}
	// The write completed; a write error means no rumble, and there is
	// nothing to report to the caller.
	s.pending = false
	return true
}

func (s *sonyRumbler) issueWrite(strong, weak byte) {
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

	copy(s.buf, report)
	var written uint32
	if err := windows.WriteFile(s.handle, s.buf, &written, &s.ov); errors.Is(err, windows.ERROR_IO_PENDING) {
		s.pending = true
	}
	// Any other write error means no rumble; there is nothing to report to
	// the caller.
}

func (s *sonyRumbler) close() {
	if s.handle != 0 {
		// Cancel any pending write before releasing the handle. CancelIo is
		// not usable here: it only cancels I/O issued by the calling thread,
		// and goroutines migrate between threads. buf stays referenced by
		// this struct, so its memory remains valid even if the cancellation
		// completes asynchronously.
		_ = windows.CancelIoEx(s.handle, nil)
		_ = windows.CloseHandle(s.handle)
		s.handle = 0
	}
	if s.ov.HEvent != 0 {
		_ = windows.CloseHandle(s.ov.HEvent)
		s.ov.HEvent = 0
	}
}

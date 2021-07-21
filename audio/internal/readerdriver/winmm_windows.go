// Copyright 2021 The Ebiten Authors
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

package readerdriver

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winmm = windows.NewLazySystemDLL("winmm")
)

var (
	procWaveOutOpen            = winmm.NewProc("waveOutOpen")
	procWaveOutClose           = winmm.NewProc("waveOutClose")
	procWaveOutPause           = winmm.NewProc("waveOutPause")
	procWaveOutPrepareHeader   = winmm.NewProc("waveOutPrepareHeader")
	procWaveOutReset           = winmm.NewProc("waveOutReset")
	procWaveOutRestart         = winmm.NewProc("waveOutRestart")
	procWaveOutUnprepareHeader = winmm.NewProc("waveOutUnprepareHeader")
	procWaveOutWrite           = winmm.NewProc("waveOutWrite")
)

type wavehdr struct {
	lpData          uintptr
	dwBufferLength  uint32
	dwBytesRecorded uint32
	dwUser          uintptr
	dwFlags         uint32
	dwLoops         uint32
	lpNext          uintptr
	reserved        uintptr
}

type waveformatex struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

const (
	waveFormatIEEEFloat = 3
	whdrInqueue         = 16
)

type mmresult uint

const (
	mmsyserrNoerror       mmresult = 0
	mmsyserrError         mmresult = 1
	mmsyserrBaddeviceid   mmresult = 2
	mmsyserrAllocated     mmresult = 4
	mmsyserrInvalidhandle mmresult = 5
	mmsyserrNodriver      mmresult = 6
	mmsyserrNomem         mmresult = 7
	waverrBadformat       mmresult = 32
	waverrStillplaying    mmresult = 33
	waverrUnprepared      mmresult = 34
	waverrSync            mmresult = 35
)

func (m mmresult) String() string {
	switch m {
	case mmsyserrNoerror:
		return "MMSYSERR_NOERROR"
	case mmsyserrError:
		return "MMSYSERR_ERROR"
	case mmsyserrBaddeviceid:
		return "MMSYSERR_BADDEVICEID"
	case mmsyserrAllocated:
		return "MMSYSERR_ALLOCATED"
	case mmsyserrInvalidhandle:
		return "MMSYSERR_INVALIDHANDLE"
	case mmsyserrNodriver:
		return "MMSYSERR_NODRIVER"
	case mmsyserrNomem:
		return "MMSYSERR_NOMEM"
	case waverrBadformat:
		return "WAVERR_BADFORMAT"
	case waverrStillplaying:
		return "WAVERR_STILLPLAYING"
	case waverrUnprepared:
		return "WAVERR_UNPREPARED"
	case waverrSync:
		return "WAVERR_SYNC"
	}
	return fmt.Sprintf("MMRESULT (%d)", m)
}

type winmmError struct {
	fname    string
	errno    windows.Errno
	mmresult mmresult
}

func (e *winmmError) Error() string {
	if e.errno != 0 {
		return fmt.Sprintf("winmm error at %s: Errno: %d", e.fname, e.errno)
	}
	if e.mmresult != mmsyserrNoerror {
		return fmt.Sprintf("winmm error at %s: %s", e.fname, e.mmresult)
	}
	return fmt.Sprintf("winmm error at %s", e.fname)
}

func waveOutOpen(f *waveformatex, callback uintptr) (uintptr, error) {
	const (
		waveMapper       = 0xffffffff
		callbackFunction = 0x30000
	)
	var w uintptr
	var fdwOpen uintptr
	if callback != 0 {
		fdwOpen |= callbackFunction
	}
	r, _, e := procWaveOutOpen.Call(uintptr(unsafe.Pointer(&w)), waveMapper, uintptr(unsafe.Pointer(f)),
		callback, 0, fdwOpen)
	runtime.KeepAlive(f)
	if e.(windows.Errno) != 0 {
		return 0, &winmmError{
			fname: "waveOutOpen",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return 0, &winmmError{
			fname:    "waveOutOpen",
			mmresult: mmresult(r),
		}
	}
	return w, nil
}

func waveOutClose(hwo uintptr) error {
	r, _, e := procWaveOutClose.Call(hwo)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutClose",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutClose",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutPause(hwo uintptr) error {
	r, _, e := procWaveOutPause.Call(hwo)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutPause",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutPause",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutPrepareHeader(hwo uintptr, pwh *wavehdr) error {
	r, _, e := procWaveOutPrepareHeader.Call(hwo, uintptr(unsafe.Pointer(pwh)), unsafe.Sizeof(wavehdr{}))
	runtime.KeepAlive(pwh)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutPrepareHeader",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutPrepareHeader",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutReset(hwo uintptr) error {
	r, _, e := procWaveOutReset.Call(hwo)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutReset",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutReset",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutRestart(hwo uintptr) error {
	r, _, e := procWaveOutRestart.Call(hwo)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutRestart",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutRestart",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutUnprepareHeader(hwo uintptr, pwh *wavehdr) error {
	r, _, e := procWaveOutUnprepareHeader.Call(hwo, uintptr(unsafe.Pointer(pwh)), unsafe.Sizeof(wavehdr{}))
	runtime.KeepAlive(pwh)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutUnprepareHeader",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutUnprepareHeader",
			mmresult: mmresult(r),
		}
	}
	return nil
}

func waveOutWrite(hwo uintptr, pwh *wavehdr) error {
	r, _, e := procWaveOutWrite.Call(hwo, uintptr(unsafe.Pointer(pwh)), unsafe.Sizeof(wavehdr{}))
	runtime.KeepAlive(pwh)
	if e.(windows.Errno) != 0 {
		return &winmmError{
			fname: "waveOutWrite",
			errno: e.(windows.Errno),
		}
	}
	if mmresult(r) != mmsyserrNoerror {
		return &winmmError{
			fname:    "waveOutWrite",
			mmresult: mmresult(r),
		}
	}
	return nil
}

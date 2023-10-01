// Copyright 2022 The Ebiten Authors
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

//go:build !android && !nintendosdk && !playstation5

package gamepad

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	_ABS_X     = 0x00
	_ABS_Y     = 0x01
	_ABS_Z     = 0x02
	_ABS_RX    = 0x03
	_ABS_RY    = 0x04
	_ABS_RZ    = 0x05
	_ABS_HAT0X = 0x10
	_ABS_HAT0Y = 0x11
	_ABS_HAT1X = 0x12
	_ABS_HAT1Y = 0x13
	_ABS_HAT2X = 0x14
	_ABS_HAT2Y = 0x15
	_ABS_HAT3Y = 0x17
	_ABS_MAX   = 0x3f
	_ABS_CNT   = _ABS_MAX + 1

	_BTN_MISC       = 0x100
	_BTN_GAMEPAD    = 0x130
	_BTN_A          = 0x130
	_BTN_B          = 0x131
	_BTN_NORTH      = 0x133
	_BTN_X          = 0x133
	_BTN_WEST       = 0x134
	_BTN_Y          = 0x134
	_BTN_TL         = 0x136
	_BTN_TR         = 0x137
	_BTN_TL2        = 0x138
	_BTN_TR2        = 0x139
	_BTN_SELECT     = 0x13a
	_BTN_START      = 0x13b
	_BTN_MODE       = 0x13c
	_BTN_THUMBL     = 0x13d
	_BTN_THUMBR     = 0x13e
	_BTN_DPAD_UP    = 0x220
	_BTN_DPAD_DOWN  = 0x221
	_BTN_DPAD_LEFT  = 0x222
	_BTN_DPAD_RIGHT = 0x223

	_IOC_NONE  = 0
	_IOC_WRITE = 1
	_IOC_READ  = 2

	_IOC_NRBITS   = 8
	_IOC_TYPEBITS = 8
	_IOC_SIZEBITS = 14
	_IOC_DIRBITS  = 2

	_IOC_NRSHIFT   = 0
	_IOC_TYPESHIFT = _IOC_NRSHIFT + _IOC_NRBITS
	_IOC_SIZESHIFT = _IOC_TYPESHIFT + _IOC_TYPEBITS
	_IOC_DIRSHIFT  = _IOC_SIZESHIFT + _IOC_SIZEBITS

	_KEY_MAX = 0x2ff
	_KEY_CNT = _KEY_MAX + 1

	_SYN_REPORT  = 0
	_SYN_DROPPED = 3
)

func _IOC(dir, typ, nr, size uint) uint {
	return dir<<_IOC_DIRSHIFT | typ<<_IOC_TYPESHIFT | nr<<_IOC_NRSHIFT | size<<_IOC_SIZESHIFT
}

func _IOR(typ, nr, size uint) uint {
	return _IOC(_IOC_READ, typ, nr, size)
}

func _EVIOCGABS(abs uint) uint {
	return _IOR('E', 0x40+abs, uint(unsafe.Sizeof(input_absinfo{})))
}

func _EVIOCGBIT(ev, len uint) uint {
	return _IOC(_IOC_READ, 'E', 0x20+ev, len)
}

func _EVIOCGID() uint {
	return _IOR('E', 0x02, uint(unsafe.Sizeof(input_id{})))
}

func _EVIOCGNAME(len uint) uint {
	return _IOC(_IOC_READ, 'E', 0x06, len)
}

type input_absinfo struct {
	value      int32
	minimum    int32
	maximum    int32
	fuzz       int32
	flat       int32
	resolution int32
}

type input_event struct {
	time  unix.Timeval
	typ   uint16
	code  uint16
	value int32
}

type input_id struct {
	bustype uint16
	vendor  uint16
	product uint16
	version uint16
}

func ioctl(fd int, request uint, ptr unsafe.Pointer) error {
	r, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(request), uintptr(ptr))
	if r < 0 {
		return unix.Errno(e)
	}
	return nil
}

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

// Package fbdev provides rendering on a Linux framebuffer device, for systems
// that have no window system.
package fbdev

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const _FBIOGET_VSCREENINFO = 0x4600

type fbBitfield struct {
	offset   uint32
	length   uint32
	msbRight uint32
}

// fbVarScreeninfo mirrors the kernel's fb_var_screeninfo. The whole layout is
// declared because the ioctl fills all of it.
type fbVarScreeninfo struct {
	xres        uint32
	yres        uint32
	xresVirtual uint32
	yresVirtual uint32
	xoffset     uint32
	yoffset     uint32

	bitsPerPixel uint32
	grayscale    uint32

	red    fbBitfield
	green  fbBitfield
	blue   fbBitfield
	transp fbBitfield

	nonstd   uint32
	activate uint32

	height uint32
	width  uint32

	accelFlags uint32

	pixclock    uint32
	leftMargin  uint32
	rightMargin uint32
	upperMargin uint32
	lowerMargin uint32
	hsyncLen    uint32
	vsyncLen    uint32
	sync        uint32
	vmode       uint32
	rotate      uint32
	colorspace  uint32
	reserved    [4]uint32
}

// Display is a framebuffer device.
type Display struct {
	width       int
	height      int
	refreshRate int

	redBits   int
	greenBits int
	blueBits  int
}

// OpenDisplay returns the framebuffer device's current mode.
//
// The device is not kept open: it is the source of the mode, while the vendor
// EGL implementation owns presentation.
func OpenDisplay() (*Display, error) {
	fd, err := unix.Open("/dev/fb0", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("fbdev: failed to open /dev/fb0: %w", err)
	}
	defer func() {
		_ = unix.Close(fd)
	}()

	var vi fbVarScreeninfo
	if _, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), _FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&vi))); e != 0 {
		return nil, fmt.Errorf("fbdev: FBIOGET_VSCREENINFO failed: %w", unix.Errno(e))
	}

	if vi.xres == 0 || vi.yres == 0 {
		return nil, fmt.Errorf("fbdev: /dev/fb0 reported an empty mode")
	}

	return &Display{
		width:       int(vi.xres),
		height:      int(vi.yres),
		refreshRate: vi.refreshRate(),
		redBits:     int(vi.red.length),
		greenBits:   int(vi.green.length),
		blueBits:    int(vi.blue.length),
	}, nil
}

// Size returns the display's size in pixels.
func (d *Display) Size() (width, height int) {
	return d.width, d.height
}

// RefreshRate returns the display's refresh rate in Hz.
func (d *Display) RefreshRate() int {
	return d.refreshRate
}

// BitsPerColor returns the display's bit depth per color channel.
func (d *Display) BitsPerColor() (red, green, blue int) {
	return d.redBits, d.greenBits, d.blueBits
}

// refreshRate returns the refresh rate in Hz, or 0 when the mode does not
// describe the timings.
func (vi *fbVarScreeninfo) refreshRate() int {
	if vi.pixclock == 0 {
		return 0
	}

	hTotal := uint64(vi.xres) + uint64(vi.leftMargin) + uint64(vi.rightMargin) + uint64(vi.hsyncLen)
	vTotal := uint64(vi.yres) + uint64(vi.upperMargin) + uint64(vi.lowerMargin) + uint64(vi.vsyncLen)
	if hTotal == 0 || vTotal == 0 {
		return 0
	}

	// pixclock is the duration of one pixel in picoseconds.
	frame := uint64(vi.pixclock) * hTotal * vTotal
	if frame == 0 {
		return 0
	}
	return int(1e12 / frame)
}

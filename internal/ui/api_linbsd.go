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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package ui

import (
	"runtime"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// This file binds the handful of libX11 and libXrandr functions that the X11
// backend needs on top of what internal/glfw already provides. The Display is
// borrowed from glfw via [glfw.GetX11Display], so no connection is opened here.
//
// C unsigned long is pointer-sized on every Unix ABI, matching Go's uint, so
// XID and Atom (both unsigned long in Xlib) are uint.
type (
	xID   = uint
	xAtom = uint
)

const xPropModeReplace = 0

// Modifier bits of the state mask XQueryPointer returns. Mod2 is where X11
// keymaps conventionally bind Num Lock.
const (
	xLockMask = 1 << 1
	xMod2Mask = 1 << 4
)

// xrrCrtcInfo mirrors the leading members of XRRCrtcInfo. Only the size fields
// are read, so the trailing members are omitted.
type xrrCrtcInfo struct {
	_         structs.HostLayout
	timestamp uint
	x         int32
	y         int32
	width     uint32
	height    uint32
}

var (
	xInternAtom     func(display uintptr, name string, onlyIfExists bool) xAtom
	xChangeProperty func(display uintptr, w xID, property, typ xAtom, format, mode int32, data unsafe.Pointer, nelements int32) int32
	xDeleteProperty func(display uintptr, w xID, property xAtom) int32
	xFlush          func(display uintptr) int32

	xrrGetScreenResourcesCurrent func(display uintptr, window xID) uintptr
	xrrGetCrtcInfo               func(display uintptr, resources uintptr, crtc xID) uintptr
	xrrFreeScreenResources       func(resources uintptr)
	xrrFreeCrtcInfo              func(crtcInfo uintptr)
)

var (
	x11Once      sync.Once
	x11Loaded    bool
	xrandrLoaded bool
)

// ensureX11 loads libX11 (and, if present, libXrandr) on first use. It reports
// whether libX11 is available; a false result indicates a pure Wayland session
// with no X server.
func ensureX11() bool {
	x11Once.Do(loadX11)
	return x11Loaded
}

func loadX11() {
	lib, err := openX11Library("libX11.so.6", "libX11.so")
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&xInternAtom, lib, "XInternAtom")
	purego.RegisterLibFunc(&xChangeProperty, lib, "XChangeProperty")
	purego.RegisterLibFunc(&xDeleteProperty, lib, "XDeleteProperty")
	purego.RegisterLibFunc(&xFlush, lib, "XFlush")
	for _, p := range []struct {
		proc *uintptr
		name string
	}{
		{&x11Procs.defaultScreen, "XDefaultScreen"},
		{&x11Procs.rootWindow, "XRootWindow"},
		{&x11Procs.queryPointer, "XQueryPointer"},
	} {
		proc, err := purego.Dlsym(lib, p.name)
		if err != nil {
			return
		}
		*p.proc = proc
	}
	x11Loaded = true

	// RandR is optional. Without it, monitor sizes fall back to the video mode.
	rlib, err := openX11Library("libXrandr.so.2", "libXrandr.so")
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&xrrGetScreenResourcesCurrent, rlib, "XRRGetScreenResourcesCurrent")
	purego.RegisterLibFunc(&xrrGetCrtcInfo, rlib, "XRRGetCrtcInfo")
	purego.RegisterLibFunc(&xrrFreeScreenResources, rlib, "XRRFreeScreenResources")
	purego.RegisterLibFunc(&xrrFreeCrtcInfo, rlib, "XRRFreeCrtcInfo")
	xrandrLoaded = true
}

func openX11Library(names ...string) (uintptr, error) {
	var firstErr error
	for _, name := range names {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			return lib, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return 0, firstErr
}

// xChangePropertyGeneric sets a window property from data. The X property
// format is 32 for uint and 8 for byte. An empty slice sets the property to
// zero elements.
func xChangePropertyGeneric[T byte | uint](display uintptr, w xID, property, typ xAtom, mode int32, data []T) int32 {
	// Xlib takes 32-bit property data as C longs, so the format does not
	// follow the element size.
	format := int32(32)
	if unsafe.Sizeof(T(0)) == 1 {
		format = 8
	}
	var head unsafe.Pointer
	if len(data) > 0 {
		head = unsafe.Pointer(&data[0])
	}
	return xChangeProperty(display, w, property, typ, format, mode, head, int32(len(data)))
}

// x11Call holds the arguments and the out-parameters of an Xlib function call.
type x11Call struct {
	args   [9]uintptr
	pinner runtime.Pinner

	ids    [2]xID
	int32s [4]int32
	uint32 uint32
}

var x11CallPool = sync.Pool{
	New: func() any {
		return &x11Call{}
	},
}

// getX11Call returns a pinned x11Call. Release it with putX11Call.
func getX11Call() *x11Call {
	c := x11CallPool.Get().(*x11Call)
	c.pinner.Pin(c)
	return c
}

func putX11Call(c *x11Call) {
	c.pinner.Unpin()
	x11CallPool.Put(c)
}

// call calls fn with args. args must not hold Go pointers unless their objects are pinned.
func (c *x11Call) call(fn uintptr, args ...uintptr) uintptr {
	n := copy(c.args[:], args)
	r, _, _ := purego.SyscallN(fn, c.args[:n]...)
	return r
}

// x11Procs holds the addresses of the Xlib functions below.
var x11Procs struct {
	defaultScreen uintptr
	rootWindow    uintptr
	queryPointer  uintptr
}

func xDefaultScreen(display uintptr) int32 {
	c := getX11Call()
	defer putX11Call(c)
	return int32(c.call(x11Procs.defaultScreen, display))
}

func xRootWindow(display uintptr, screen int32) xID {
	c := getX11Call()
	defer putX11Call(c)
	return xID(c.call(x11Procs.rootWindow, display, uintptr(screen)))
}

func xQueryPointer(display uintptr, w xID, rootReturn, childReturn *xID, rootXReturn, rootYReturn, winXReturn, winYReturn *int32, maskReturn *uint32) bool {
	c := getX11Call()
	defer putX11Call(c)
	c.ids[0], c.ids[1] = *rootReturn, *childReturn
	c.int32s[0], c.int32s[1], c.int32s[2], c.int32s[3] = *rootXReturn, *rootYReturn, *winXReturn, *winYReturn
	c.uint32 = *maskReturn
	r := c.call(x11Procs.queryPointer, display, uintptr(w), uintptr(unsafe.Pointer(&c.ids[0])), uintptr(unsafe.Pointer(&c.ids[1])),
		uintptr(unsafe.Pointer(&c.int32s[0])), uintptr(unsafe.Pointer(&c.int32s[1])), uintptr(unsafe.Pointer(&c.int32s[2])), uintptr(unsafe.Pointer(&c.int32s[3])), uintptr(unsafe.Pointer(&c.uint32)))
	*rootReturn, *childReturn = c.ids[0], c.ids[1]
	*rootXReturn, *rootYReturn, *winXReturn, *winYReturn = c.int32s[0], c.int32s[1], c.int32s[2], c.int32s[3]
	*maskReturn = c.uint32
	return byte(r) != 0
}

func x11RootWindow(display uintptr) xID {
	return xRootWindow(display, xDefaultScreen(display))
}

// x11QueryPointer returns the cursor position relative to the root window, and
// the modifier and button state mask. ok is false when the pointer is on
// another screen.
func x11QueryPointer(display uintptr) (x, y int, mask uint32, ok bool) {
	var (
		rootReturn, childReturn  xID
		rootX, rootY, winX, winY int32
	)
	if !xQueryPointer(display, x11RootWindow(display), &rootReturn, &childReturn, &rootX, &rootY, &winX, &winY, &mask) {
		return 0, 0, 0, false
	}
	return int(rootX), int(rootY), mask, true
}

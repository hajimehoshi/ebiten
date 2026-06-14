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

// xrrCrtcInfo mirrors the leading members of XRRCrtcInfo. Only the size fields
// are read, so the trailing members are omitted.
type xrrCrtcInfo struct {
	timestamp uint
	x         int32
	y         int32
	width     uint32
	height    uint32
}

var (
	xDefaultScreen  func(display uintptr) int32
	xRootWindow     func(display uintptr, screen int32) xID
	xInternAtom     func(display uintptr, name string, onlyIfExists bool) xAtom
	xChangeProperty func(display uintptr, w xID, property, typ xAtom, format, mode int32, data unsafe.Pointer, nelements int32) int32
	xDeleteProperty func(display uintptr, w xID, property xAtom) int32
	xQueryPointer   func(display uintptr, w xID, rootReturn, childReturn *xID, rootXReturn, rootYReturn, winXReturn, winYReturn *int32, maskReturn *uint32) bool
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
	purego.RegisterLibFunc(&xDefaultScreen, lib, "XDefaultScreen")
	purego.RegisterLibFunc(&xRootWindow, lib, "XRootWindow")
	purego.RegisterLibFunc(&xInternAtom, lib, "XInternAtom")
	purego.RegisterLibFunc(&xChangeProperty, lib, "XChangeProperty")
	purego.RegisterLibFunc(&xDeleteProperty, lib, "XDeleteProperty")
	purego.RegisterLibFunc(&xQueryPointer, lib, "XQueryPointer")
	purego.RegisterLibFunc(&xFlush, lib, "XFlush")
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

func x11RootWindow(display uintptr) xID {
	return xRootWindow(display, xDefaultScreen(display))
}

// x11QueryPointerPosition returns the cursor position relative to the root
// window. ok is false when the pointer is on another screen.
func x11QueryPointerPosition(display uintptr) (x, y int, ok bool) {
	var (
		rootReturn, childReturn  xID
		rootX, rootY, winX, winY int32
		mask                     uint32
	)
	if !xQueryPointer(display, x11RootWindow(display), &rootReturn, &childReturn, &rootX, &rootY, &winX, &winY, &mask) {
		return 0, 0, false
	}
	return int(rootX), int(rootY), true
}

// x11CrtcSize returns the pixel size of the given CRTC. ok is false when RandR
// is unavailable or the CRTC cannot be queried.
func x11CrtcSize(display uintptr, crtc xID) (width, height int, ok bool) {
	if !xrandrLoaded {
		return 0, 0, false
	}
	resources := xrrGetScreenResourcesCurrent(display, x11RootWindow(display))
	if resources == 0 {
		return 0, 0, false
	}
	defer xrrFreeScreenResources(resources)

	info := xrrGetCrtcInfo(display, resources, crtc)
	if info == 0 {
		return 0, 0, false
	}
	defer xrrFreeCrtcInfo(info)

	ci := (*xrrCrtcInfo)(unsafe.Pointer(info))
	return int(ci.width), int(ci.height), true
}

// x11SetWindowThemeVariant sets the _GTK_THEME_VARIANT property on the window,
// or removes it when variant is empty.
func x11SetWindowThemeVariant(display, window uintptr, variant string) {
	gtkThemeVariant := xInternAtom(display, "_GTK_THEME_VARIANT", false)
	if variant == "" {
		_ = xDeleteProperty(display, xID(window), gtkThemeVariant)
	} else {
		utf8String := xInternAtom(display, "UTF8_STRING", false)
		data := []byte(variant)
		_ = xChangeProperty(display, xID(window), gtkThemeVariant, utf8String, 8, xPropModeReplace, unsafe.Pointer(&data[0]), int32(len(data)))
	}
	// Properties are buffered until the next round-trip; flush so the change
	// takes effect immediately.
	xFlush(display)
}

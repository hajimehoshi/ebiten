// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

import (
	"golang.org/x/sys/windows"
)

const (
	_GLFW_WNDCLASSNAME = "GLFW30"
)

type platformWindowState struct {
	handle    windows.HWND
	bigIcon   _HICON
	smallIcon _HICON

	cursorTracked  bool
	frameAction    bool
	iconified      bool
	maximized      bool
	transparent    bool // Whether to enable framebuffer transparency on DWM
	scaleToMonitor bool

	// Cached size used to filter out duplicate events
	width  int
	height int

	// The last received cursor position, regardless of source
	lastCursorPosX int
	lastCursorPosY int

	// The last recevied high surrogate when decoding pairs of UTF-16 messages
	highSurrogate uint16
}

type platformMonitorState struct {
	handle _HMONITOR

	// This size matches the static size of DISPLAY_DEVICE.DeviceName
	adapterName string
	displayName string
	modesPruned bool
	modeChanged bool
}

type platformCursorState struct {
	handle _HCURSOR
}

type platformTLSState struct {
	allocated bool
	index     uint32
}

type platformLibraryWindowState struct {
	instance                 _HINSTANCE
	helperWindowHandle       windows.HWND
	deviceNotificationHandle _HDEVNOTIFY
	acquiredMonitorCount     int
	clipboardString          string
	keycodes                 [512]Key
	scancodes                [KeyLast + 1]int
	keynames                 [KeyLast + 1]string

	// Where to place the cursor when re-enabled
	restoreCursorPosX float64
	restoreCursorPosY float64

	// The window whose disabled cursor mode is active
	disabledCursorWindow *Window
	rawInput             []byte
	mouseTrailSize       uint32
}

// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

import "golang.org/x/sys/windows"

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

type platformContextState struct {
	dc       _HDC
	handle   _HGLRC
	interval int
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

type platformLibraryContextState struct {
	inited bool

	EXT_swap_control               bool
	EXT_colorspace                 bool
	ARB_multisample                bool
	ARB_framebuffer_sRGB           bool
	EXT_framebuffer_sRGB           bool
	ARB_pixel_format               bool
	ARB_create_context             bool
	ARB_create_context_profile     bool
	EXT_create_context_es2_profile bool
	ARB_create_context_robustness  bool
	ARB_create_context_no_error    bool
	ARB_context_flush_control      bool
}

func _IsWindowsXPOrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WINXP)), uint16(_LOBYTE(_WIN32_WINNT_WINXP)), 0)
}

func _IsWindowsVistaOrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_VISTA)), uint16(_LOBYTE(_WIN32_WINNT_VISTA)), 0)
}

func _IsWindows7OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WIN7)), uint16(_LOBYTE(_WIN32_WINNT_WIN7)), 0)
}

func _IsWindows8OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WIN8)), uint16(_LOBYTE(_WIN32_WINNT_WIN8)), 0)
}

func _IsWindows8Point1OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WINBLUE)), uint16(_LOBYTE(_WIN32_WINNT_WINBLUE)), 0)
}

func isWindows10AnniversaryUpdateOrGreaterWin32() bool {
	return isWindows10BuildOrGreaterWin32(14393)
}

func isWindows10CreatorsUpdateOrGreaterWin32() bool {
	return isWindows10BuildOrGreaterWin32(15063)
}

// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfwwin

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_GLFW_INSERT_FIRST = 0
	_GLFW_INSERT_LAST  = 1
)

var _glfw library

type initconfig struct {
	hatButtons bool
}

type wndconfig struct {
	width          int
	height         int
	title          string
	resizable      bool
	visible        bool
	decorated      bool
	focused        bool
	autoIconify    bool
	floating       bool
	maximized      bool
	centerCursor   bool
	focusOnShow    bool
	scaleToMonitor bool
}

type ctxconfig struct {
	client     int
	source     int
	major      int
	minor      int
	forward    bool
	debug      bool
	noerror    bool
	profile    int
	robustness int
	release    int
	share      *Window
}

type fbconfig struct {
	redBits        int
	greenBits      int
	blueBits       int
	alphaBits      int
	depthBits      int
	stencilBits    int
	accumRedBits   int
	accumGreenBits int
	accumBlueBits  int
	accumAlphaBits int
	auxBuffers     int
	stereo         bool
	samples        int
	sRGB           bool
	doublebuffer   bool
	transparent    bool
	handle         uintptr
}

type context struct {
	client     int
	source     int
	major      int
	minor      int
	revision   int
	forward    bool
	debug      bool
	noerror    bool
	profile    int
	robustness int
	release    int

	// TODO: Put these functions in an interface type.
	makeCurrent        func(*Window) error
	swapBuffers        func(*Window) error
	swapInterval       func(int) error
	extensionSupported func(string) bool
	getProcAddress     func(string) uintptr
	destroy            func(*Window) error

	wgl struct {
		dc       _HDC
		handle   _HGLRC
		interval int
	}
}

type (
	PosCallback             func(w *Window, xpos int, ypos int)
	SizeCallback            func(w *Window, width int, height int)
	CloseCallback           func(w *Window)
	RefreshCallback         func(w *Window)
	FocusCallback           func(w *Window, focused bool)
	IconifyCallback         func(w *Window, iconified bool)
	MaximizeCallback        func(w *Window, iconified bool)
	FramebufferSizeCallback func(w *Window, width int, height int)
	ContentScaleCallback    func(w *Window, x float32, y float32)
	MouseButtonCallback     func(w *Window, button MouseButton, action Action, mods ModifierKey)
	CursorPosCallback       func(w *Window, xpos float64, ypos float64)
	CursorEnterCallback     func(w *Window, entered bool)
	ScrollCallback          func(w *Window, xoff float64, yoff float64)
	KeyCallback             func(w *Window, key Key, scancode int, action Action, mods ModifierKey)
	CharCallback            func(w *Window, char rune)
	CharModsCallback        func(w *Window, char rune, mods ModifierKey)
	DropCallback            func(w *Window, names []string)
	MonitorCallback         func(monitor *Monitor, event PeripheralEvent)
)

type Window struct {
	resizable    bool
	decorated    bool
	autoIconify  bool
	floating     bool
	focusOnShow  bool
	shouldClose  bool
	userPointer  unsafe.Pointer
	doublebuffer bool
	videoMode    VidMode
	monitor      *Monitor
	cursor       *Cursor

	minwidth  int
	minheight int
	maxwidth  int
	maxheight int
	numer     int
	denom     int

	stickyKeys         bool
	stickyMouseButtons bool
	lockKeyMods        bool
	cursorMode         int
	mouseButtons       [MouseButtonLast + 1]Action
	keys               [KeyLast + 1]Action
	// Virtual cursor position when cursor is disabled
	virtualCursorPosX float64
	virtualCursorPosY float64
	rawMouseMotion    bool

	context context

	callbacks struct {
		pos         PosCallback
		size        SizeCallback
		close       CloseCallback
		refresh     RefreshCallback
		focus       FocusCallback
		iconify     IconifyCallback
		maximize    MaximizeCallback
		fbsize      FramebufferSizeCallback
		scale       ContentScaleCallback
		mouseButton MouseButtonCallback
		cursorPos   CursorPosCallback
		cursorEnter CursorEnterCallback
		scroll      ScrollCallback
		key         KeyCallback
		character   CharCallback
		charmods    CharModsCallback
		drop        DropCallback
	}

	win32 struct {
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
}

type Monitor struct {
	name string

	window *Window

	modes []*VidMode

	win32 struct {
		handle _HMONITOR

		// This size matches the static size of DISPLAY_DEVICE.DeviceName
		adapterName string
		displayName string
		modesPruned bool
		modeChanged bool
	}
}

type Cursor struct {
	win32 struct {
		handle _HCURSOR
	}
}

type tls struct {
	win32 struct {
		allocated bool
		index     uint32
	}
}

type library struct {
	initialized bool

	hints struct {
		init        initconfig
		framebuffer fbconfig
		window      wndconfig
		context     ctxconfig
		refreshRate int
	}

	errors  []error // TODO: Check the error at polling?
	cursors []*Cursor
	windows []*Window

	monitors []*Monitor

	contextSlot tls

	callbacks struct {
		monitor MonitorCallback
	}

	win32 struct {
		instance                 _HINSTANCE
		helperWindowHandle       windows.HWND
		deviceNotificationHandle _HDEVNOTIFY
		acquiredMonitorCount     int
		clipboardString          string
		keycodes                 [512]Key
		scancodes                [KeyLast + 1]int

		// Where to place the cursor when re-enabled
		restoreCursorPosX float64
		restoreCursorPosY float64

		// The window whose disabled cursor mode is active
		disabledCursorWindow *Window
		rawInput             []byte
		mouseTrailSize       uint32
	}

	wgl struct {
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
}

func boolToInt(x bool) int {
	if x {
		return 1
	}
	return 0
}

func intToBool(x int) bool {
	return x != 0
}

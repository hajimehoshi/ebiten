// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

import (
	"sync/atomic"
)

// This file is the Go translation of x11_platform_linbsd.h: the X11-specific
// platform state embedded in the shared core's Window, Monitor, Cursor, and
// library structs. The extension function fields are registered from the
// dlopened extension libraries during platformInit, mirroring the C
// implementation.

type platformWindowState struct {
	colormap _XID
	handle   _XID
	parent   _XID
	ic       uintptr // XIC

	overrideRedirect bool
	iconified        bool
	maximized        bool

	// Whether the visual supports framebuffer transparency
	transparent bool

	// Cached position and size used to filter out duplicate events
	width, height int
	xpos, ypos    int

	// The last received cursor position, regardless of source
	lastCursorPosX, lastCursorPosY int
	// The last position the cursor was warped to by GLFW
	warpCursorPosX, warpCursorPosY int

	// The time of the last KeyPress event per keycode, for discarding
	// duplicate key events generated for some keys by ibus
	keyPressTimes [256]_Time

	// _NET_WM_SYNC_REQUEST frame synchronization counter (0 when unsupported).
	// It is set once at window creation and cleared at destruction, both on the
	// main thread outside the game loop, so it needs no synchronization.
	syncCounter _XID
	// syncValue is the counter value the window manager most recently requested
	// (packed as hi<<32|lo), to be set on syncCounter after the next buffer swap.
	// syncRequested reports whether such a request is awaiting acknowledgment.
	// Both are written on the main thread (event handling) and read on the render
	// thread (buffer swap, which runs asynchronously when vsync is off), so they
	// are accessed atomically.
	syncValue     atomic.Uint64
	syncRequested atomic.Bool
}

type platformMonitorState struct {
	output  _RROutput
	crtc    _RRCrtc
	oldMode _RRMode

	// Index of corresponding Xinerama screen, for EWMH full screen window
	// placement
	index int
}

type platformCursorState struct {
	handle _XID
}

type platformLibraryWindowState struct {
	display uintptr // Display*
	screen  int
	root    _XID

	// System content scale
	contentScaleX, contentScaleY float32
	// Helper window for IPC
	helperWindowHandle _XID
	// Invisible cursor for hidden cursor mode
	hiddenCursorHandle _XID
	// XID to *Window mapping (replaces the C implementation's XContext)
	windowsByXID map[_XID]*Window
	// XIM input method
	im uintptr
	// The previous X error handler, to be restored later
	errorHandler uintptr
	// Most recent error code received by X error handler
	errorCode int
	// Primary selection string (while the primary selection is owned)
	primarySelectionString string
	// Clipboard string (while the selection is owned)
	clipboardString string
	// Key name string
	keynames [KeyLast + 1]string
	// X11 keycode to GLFW key LUT
	keycodes [256]Key
	// GLFW key to X11 keycode LUT
	scancodes [KeyLast + 1]int
	// Where to place the cursor when re-enabled
	restoreCursorPosX, restoreCursorPosY float64
	// The window whose disabled cursor mode is active
	disabledCursorWindow *Window
	emptyEventPipe       [2]int

	// Window manager atoms
	NET_SUPPORTED                  _Atom
	NET_SUPPORTING_WM_CHECK        _Atom
	WM_PROTOCOLS                   _Atom
	WM_STATE                       _Atom
	WM_DELETE_WINDOW               _Atom
	NET_WM_NAME                    _Atom
	NET_WM_ICON_NAME               _Atom
	NET_WM_ICON                    _Atom
	NET_WM_PID                     _Atom
	NET_WM_PING                    _Atom
	NET_WM_WINDOW_TYPE             _Atom
	NET_WM_WINDOW_TYPE_NORMAL      _Atom
	NET_WM_STATE                   _Atom
	NET_WM_STATE_ABOVE             _Atom
	NET_WM_STATE_FULLSCREEN        _Atom
	NET_WM_STATE_MAXIMIZED_VERT    _Atom
	NET_WM_STATE_MAXIMIZED_HORZ    _Atom
	NET_WM_STATE_DEMANDS_ATTENTION _Atom
	NET_WM_BYPASS_COMPOSITOR       _Atom
	NET_WM_FULLSCREEN_MONITORS     _Atom
	NET_WM_WINDOW_OPACITY          _Atom
	NET_WM_SYNC_REQUEST            _Atom
	NET_WM_SYNC_REQUEST_COUNTER    _Atom
	NET_WM_CM_Sx                   _Atom
	NET_WORKAREA                   _Atom
	NET_CURRENT_DESKTOP            _Atom
	NET_ACTIVE_WINDOW              _Atom
	NET_FRAME_EXTENTS              _Atom
	NET_REQUEST_FRAME_EXTENTS      _Atom
	MOTIF_WM_HINTS                 _Atom

	// Xdnd (drag and drop) atoms
	XdndAware      _Atom
	XdndEnter      _Atom
	XdndPosition   _Atom
	XdndStatus     _Atom
	XdndActionCopy _Atom
	XdndDrop       _Atom
	XdndFinished   _Atom
	XdndSelection  _Atom
	XdndTypeList   _Atom
	text_uri_list  _Atom

	// Selection (clipboard) atoms
	TARGETS           _Atom
	MULTIPLE          _Atom
	INCR              _Atom
	CLIPBOARD         _Atom
	PRIMARY           _Atom
	CLIPBOARD_MANAGER _Atom
	SAVE_TARGETS      _Atom
	NULL_             _Atom
	UTF8_STRING       _Atom
	COMPOUND_STRING   _Atom
	ATOM_PAIR         _Atom
	GLFW_SELECTION    _Atom

	randr struct {
		available     bool
		handle        uintptr
		eventBase     int32
		errorBase     int32
		major         int32
		minor         int32
		monitorBroken bool

		FreeCrtcInfo              func(crtcInfo uintptr)
		FreeOutputInfo            func(outputInfo uintptr)
		FreeScreenResources       func(resources uintptr)
		GetCrtcInfo               func(display uintptr, resources uintptr, crtc _RRCrtc) uintptr
		GetOutputInfo             func(display uintptr, resources uintptr, output _RROutput) uintptr
		GetOutputPrimary          func(display uintptr, window _XID) _RROutput
		GetScreenResourcesCurrent func(display uintptr, window _XID) uintptr
		QueryExtension            func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		QueryVersion              func(display uintptr, majorReturn, minorReturn *int32) int32
		SelectInput               func(display uintptr, window _XID, mask int32)
		SetCrtcConfig             func(display uintptr, resources uintptr, crtc _RRCrtc, timestamp _Time, x, y int32, mode _RRMode, rotation _Rotation, outputs *_RROutput, noutputs int32) int32
		UpdateConfiguration       func(event *_XEvent) int32
	}

	xkb struct {
		available   bool
		detectable  bool
		majorOpcode int32
		eventBase   int32
		errorBase   int32
		major       int32
		minor       int32
		group       uint32
	}

	saver struct {
		count    int32
		timeout  int32
		interval int32
		blanking int32
		exposure int32
	}

	xdnd struct {
		version int
		source  _XID
		format  _Atom
	}

	xcursor struct {
		handle uintptr

		ImageCreate      func(width, height int32) uintptr
		ImageDestroy     func(image uintptr)
		ImageLoadCursor  func(display uintptr, image uintptr) _XID
		GetTheme         func(display uintptr) uintptr
		GetDefaultSize   func(display uintptr) int32
		LibraryLoadImage func(library string, theme uintptr, size int32) uintptr
	}

	xinerama struct {
		available bool
		handle    uintptr
		major     int32
		minor     int32

		IsActive       func(display uintptr) bool
		QueryExtension func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		QueryScreens   func(display uintptr, numberReturn *int32) uintptr
	}

	xi struct {
		available   bool
		handle      uintptr
		majorOpcode int32
		eventBase   int32
		errorBase   int32
		major       int32
		minor       int32

		QueryVersion func(display uintptr, majorVersionInOut, minorVersionInOut *int32) int32
		SelectEvents func(display uintptr, window _XID, masks *_XIEventMask, numMasks int32) int32
	}

	xrender struct {
		available bool
		handle    uintptr
		major     int32
		minor     int32
		eventBase int32
		errorBase int32

		QueryExtension   func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		QueryVersion     func(display uintptr, majorReturn, minorReturn *int32) int32
		FindVisualFormat func(display uintptr, visual uintptr) uintptr
	}

	xshape struct {
		available bool
		handle    uintptr
		major     int32
		minor     int32
		eventBase int32
		errorBase int32

		QueryExtension func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		QueryVersion   func(display uintptr, majorVersionReturn, minorVersionReturn *int32) int32
		CombineRegion  func(display uintptr, window _XID, destKind int32, xOff, yOff int32, region _Region, op int32)
		CombineMask    func(display uintptr, window _XID, destKind int32, xOff, yOff int32, src _XID, op int32)
	}

	// xsync holds the X Sync extension entry points used for _NET_WM_SYNC_REQUEST
	// frame synchronization. The counter value, an XSyncValue struct, is passed as
	// a single machine word rather than by value, which is ABI-correct only on
	// 64-bit architectures (see initExtensions), so available is false on 32-bit
	// ones.
	xsync struct {
		available bool
		major     int32
		minor     int32
		eventBase int32
		errorBase int32

		QueryExtension func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		Initialize     func(display uintptr, majorReturn, minorReturn *int32) int32
		CreateCounter  func(display uintptr, initialValue uint64) _XID
		SetCounter     func(display uintptr, counter _XID, value uint64) int32
		DestroyCounter func(display uintptr, counter _XID) int32
	}
}

// platformContextState and platformLibraryContextState hold the GLX and EGL
// context state.
type platformContextState struct {
	glx struct {
		handle uintptr // GLXContext
		window _XID    // GLXWindow
	}
	egl struct {
		config  uintptr // EGLConfig
		handle  uintptr // EGLContext
		surface uintptr // EGLSurface
		client  uintptr // dlopened client library
	}
}

type platformLibraryContextState struct {
	glx struct {
		major, minor int32
		eventBase    int32
		errorBase    int32
		handle       uintptr

		SGI_swap_control               bool
		EXT_swap_control               bool
		MESA_swap_control              bool
		ARB_multisample                bool
		ARB_framebuffer_sRGB           bool
		EXT_framebuffer_sRGB           bool
		ARB_create_context             bool
		ARB_create_context_profile     bool
		ARB_create_context_robustness  bool
		EXT_create_context_es2_profile bool
		ARB_create_context_no_error    bool
		ARB_context_flush_control      bool

		GetFBConfigs          func(display uintptr, screen int32, nelements *int32) uintptr
		GetFBConfigAttrib     func(display uintptr, config uintptr, attribute int32, value *int32) int32
		GetClientString       func(display uintptr, name int32) uintptr
		QueryExtension        func(display uintptr, errorBase, eventBase *int32) bool
		QueryVersion          func(display uintptr, major, minor *int32) bool
		DestroyContext        func(display uintptr, ctx uintptr)
		MakeCurrent           func(display uintptr, drawable _XID, ctx uintptr) bool
		SwapBuffers           func(display uintptr, drawable _XID)
		QueryExtensionsString func(display uintptr, screen int32) uintptr
		CreateNewContext      func(display uintptr, config uintptr, renderType int32, share uintptr, direct bool) uintptr
		CreateWindow          func(display uintptr, config uintptr, win _XID, attribs *int32) _XID
		DestroyWindow         func(display uintptr, window _XID)
		GetVisualFromFBConfig func(display uintptr, config uintptr) uintptr

		GetProcAddress    func(procname string) uintptr
		GetProcAddressARB func(procname string) uintptr

		SwapIntervalSGI         func(interval int32) int32
		SwapIntervalEXT         func(display uintptr, drawable _XID, interval int32)
		SwapIntervalMESA        func(interval int32) int32
		CreateContextAttribsARB func(display uintptr, config uintptr, share uintptr, direct bool, attribs *int32) uintptr
	}
	egl struct {
		major, minor int32
		handle       uintptr
		display      uintptr
		prefix       bool

		KHR_create_context          bool
		KHR_create_context_no_error bool
		KHR_gl_colorspace           bool
		KHR_get_all_proc_addresses  bool
		KHR_context_flush_control   bool
		EXT_present_opaque          bool

		GetConfigAttrib     func(display uintptr, config uintptr, attribute int32, value *int32) bool
		GetConfigs          func(display uintptr, configs *uintptr, configSize int32, numConfig *int32) bool
		GetDisplay          func(displayID uintptr) uintptr
		GetError            func() int32
		Initialize          func(display uintptr, major, minor *int32) bool
		Terminate           func(display uintptr) bool
		BindAPI             func(api int32) bool
		CreateContext       func(display uintptr, config uintptr, shareContext uintptr, attribList *int32) uintptr
		DestroySurface      func(display uintptr, surface uintptr) bool
		DestroyContext      func(display uintptr, ctx uintptr) bool
		CreateWindowSurface func(display uintptr, config uintptr, win _XID, attribList *int32) uintptr
		MakeCurrent         func(display uintptr, draw uintptr, read uintptr, ctx uintptr) bool
		SwapBuffers         func(display uintptr, surface uintptr) bool
		SwapInterval        func(display uintptr, interval int32) bool
		QueryString         func(display uintptr, name int32) uintptr
		GetProcAddress      func(procname string) uintptr
	}
}

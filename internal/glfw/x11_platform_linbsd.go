// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

// This file is the Go translation of x11_platform_linbsd.h: the X11-specific
// platform state embedded in the shared core's Window, Monitor, Cursor, and
// library structs. The extension function fields are registered from the
// dlopened extension libraries during platformInit, mirroring the C
// implementation.

type platformWindowState struct {
	colormap XID
	handle   XID
	parent   XID
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
	keyPressTimes [256]Time
}

type platformMonitorState struct {
	output  RROutput
	crtc    RRCrtc
	oldMode RRMode

	// Index of corresponding Xinerama screen, for EWMH full screen window
	// placement
	index int
}

type platformCursorState struct {
	handle XID
}

type platformLibraryWindowState struct {
	display uintptr // Display*
	screen  int
	root    XID

	// System content scale
	contentScaleX, contentScaleY float32
	// Helper window for IPC
	helperWindowHandle XID
	// Invisible cursor for hidden cursor mode
	hiddenCursorHandle XID
	// XID to *Window mapping (replaces the C implementation's XContext)
	windowsByXID map[XID]*Window
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
	NET_SUPPORTED                  Atom
	NET_SUPPORTING_WM_CHECK        Atom
	WM_PROTOCOLS                   Atom
	WM_STATE                       Atom
	WM_DELETE_WINDOW               Atom
	NET_WM_NAME                    Atom
	NET_WM_ICON_NAME               Atom
	NET_WM_ICON                    Atom
	NET_WM_PID                     Atom
	NET_WM_PING                    Atom
	NET_WM_WINDOW_TYPE             Atom
	NET_WM_WINDOW_TYPE_NORMAL      Atom
	NET_WM_STATE                   Atom
	NET_WM_STATE_ABOVE             Atom
	NET_WM_STATE_FULLSCREEN        Atom
	NET_WM_STATE_MAXIMIZED_VERT    Atom
	NET_WM_STATE_MAXIMIZED_HORZ    Atom
	NET_WM_STATE_DEMANDS_ATTENTION Atom
	NET_WM_BYPASS_COMPOSITOR       Atom
	NET_WM_FULLSCREEN_MONITORS     Atom
	NET_WM_WINDOW_OPACITY          Atom
	NET_WM_CM_Sx                   Atom
	NET_WORKAREA                   Atom
	NET_CURRENT_DESKTOP            Atom
	NET_ACTIVE_WINDOW              Atom
	NET_FRAME_EXTENTS              Atom
	NET_REQUEST_FRAME_EXTENTS      Atom
	MOTIF_WM_HINTS                 Atom

	// Xdnd (drag and drop) atoms
	XdndAware      Atom
	XdndEnter      Atom
	XdndPosition   Atom
	XdndStatus     Atom
	XdndActionCopy Atom
	XdndDrop       Atom
	XdndFinished   Atom
	XdndSelection  Atom
	XdndTypeList   Atom
	text_uri_list  Atom

	// Selection (clipboard) atoms
	TARGETS           Atom
	MULTIPLE          Atom
	INCR              Atom
	CLIPBOARD         Atom
	PRIMARY           Atom
	CLIPBOARD_MANAGER Atom
	SAVE_TARGETS      Atom
	NULL_             Atom
	UTF8_STRING       Atom
	COMPOUND_STRING   Atom
	ATOM_PAIR         Atom
	GLFW_SELECTION    Atom

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
		GetCrtcInfo               func(display uintptr, resources uintptr, crtc RRCrtc) uintptr
		GetOutputInfo             func(display uintptr, resources uintptr, output RROutput) uintptr
		GetOutputPrimary          func(display uintptr, window XID) RROutput
		GetScreenResourcesCurrent func(display uintptr, window XID) uintptr
		QueryExtension            func(display uintptr, eventBaseReturn, errorBaseReturn *int32) bool
		QueryVersion              func(display uintptr, majorReturn, minorReturn *int32) int32
		SelectInput               func(display uintptr, window XID, mask int32)
		SetCrtcConfig             func(display uintptr, resources uintptr, crtc RRCrtc, timestamp Time, x, y int32, mode RRMode, rotation Rotation, outputs *RROutput, noutputs int32) int32
		UpdateConfiguration       func(event *XEvent) int32
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
		source  XID
		format  Atom
	}

	xcursor struct {
		handle uintptr

		ImageCreate      func(width, height int32) uintptr
		ImageDestroy     func(image uintptr)
		ImageLoadCursor  func(display uintptr, image uintptr) XID
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
		SelectEvents func(display uintptr, window XID, masks *XIEventMask, numMasks int32) int32
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
		CombineRegion  func(display uintptr, window XID, destKind int32, xOff, yOff int32, region Region, op int32)
		CombineMask    func(display uintptr, window XID, destKind int32, xOff, yOff int32, src XID, op int32)
	}
}

// platformContextState and platformLibraryContextState hold the GLX and EGL
// context state.
type platformContextState struct {
	glx struct {
		handle uintptr // GLXContext
		window XID     // GLXWindow
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
		MakeCurrent           func(display uintptr, drawable XID, ctx uintptr) bool
		SwapBuffers           func(display uintptr, drawable XID)
		QueryExtensionsString func(display uintptr, screen int32) uintptr
		CreateNewContext      func(display uintptr, config uintptr, renderType int32, share uintptr, direct bool) uintptr
		CreateWindow          func(display uintptr, config uintptr, win XID, attribs *int32) XID
		DestroyWindow         func(display uintptr, window XID)
		GetVisualFromFBConfig func(display uintptr, config uintptr) uintptr

		GetProcAddress    func(procname string) uintptr
		GetProcAddressARB func(procname string) uintptr

		SwapIntervalSGI         func(interval int32) int32
		SwapIntervalEXT         func(display uintptr, drawable XID, interval int32)
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
		CreateWindowSurface func(display uintptr, config uintptr, win XID, attribList *int32) uintptr
		MakeCurrent         func(display uintptr, draw uintptr, read uintptr, ctx uintptr) bool
		SwapBuffers         func(display uintptr, surface uintptr) bool
		SwapInterval        func(display uintptr, interval int32) bool
		QueryString         func(display uintptr, name int32) uintptr
		GetProcAddress      func(procname string) uintptr
	}
}

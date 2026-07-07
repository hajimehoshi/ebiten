// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

import (
	"fmt"
	"image"
	"math"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/cpu"
	"golang.org/x/sys/unix"
)

const (
	_NET_WM_STATE_REMOVE = 0
	_NET_WM_STATE_ADD    = 1

	_MWM_HINTS_FUNCTIONS = 1
	_MWM_FUNC_RESIZE     = 2
	_MWM_FUNC_MOVE       = 4
	_MWM_FUNC_MINIMIZE   = 8
	_MWM_FUNC_MAXIMIZE   = 16
	_MWM_FUNC_CLOSE      = 32

	_MWM_HINTS_DECORATIONS = 2
	_MWM_DECOR_BORDER      = 2
	_MWM_DECOR_RESIZEH     = 4
	_MWM_DECOR_TITLE       = 8
	_MWM_DECOR_MENU        = 16
	_MWM_DECOR_MINIMIZE    = 32
	_MWM_DECOR_MAXIMIZE    = 64

	_GLFW_XDND_VERSION = 5
)

// waitForData waits for data to arrive on any of the specified file
// descriptors.
func waitForData(fds []unix.PollFd, timeout *float64) bool {
	for {
		if timeout != nil {
			base := time.Now()

			milliseconds := int(*timeout * 1e3)
			result, err := unix.Poll(fds, milliseconds)

			*timeout -= time.Since(base).Seconds()

			if result > 0 {
				return true
			}
			if err != nil && err != unix.EINTR && err != unix.EAGAIN {
				return false
			}
			if *timeout <= 0 {
				return false
			}
		} else {
			result, err := unix.Poll(fds, -1)
			if result > 0 {
				return true
			}
			if err != nil && err != unix.EINTR && err != unix.EAGAIN {
				return false
			}
		}
	}
}

// waitForX11Event waits for event data to arrive on the X11 display socket.
func waitForX11Event(timeout *float64) bool {
	fds := []unix.PollFd{
		{Fd: int32(xConnectionNumber(_glfw.platformWindow.display)), Events: unix.POLLIN},
	}

	for xPending(_glfw.platformWindow.display) == 0 {
		if !waitForData(fds, timeout) {
			return false
		}
	}

	return true
}

// waitForAnyEvent waits for event data to arrive on any event file
// descriptor.
func waitForAnyEvent(timeout *float64) bool {
	fds := []unix.PollFd{
		{Fd: int32(xConnectionNumber(_glfw.platformWindow.display)), Events: unix.POLLIN},
		{Fd: int32(_glfw.platformWindow.emptyEventPipe[0]), Events: unix.POLLIN},
	}

	for xPending(_glfw.platformWindow.display) == 0 {
		if !waitForData(fds, timeout) {
			return false
		}

		for _, fd := range fds[1:] {
			if fd.Revents&unix.POLLIN != 0 {
				return true
			}
		}
	}

	return true
}

// writeEmptyEvent writes a byte to the empty event pipe.
func writeEmptyEvent() {
	for {
		b := []byte{0}
		result, err := unix.Write(_glfw.platformWindow.emptyEventPipe[1], b)
		if result == 1 || err != unix.EINTR {
			break
		}
	}
}

// drainEmptyEvents drains available data from the empty event pipe.
func drainEmptyEvents() {
	for {
		dummy := make([]byte, 64)
		_, err := unix.Read(_glfw.platformWindow.emptyEventPipe[0], dummy)
		if err != nil && err != unix.EINTR {
			break
		}
	}
}

// waitForVisibilityNotify waits until a VisibilityNotify event arrives for
// the specified window or the timeout period elapses (ICCCM section 4.2.2).
func waitForVisibilityNotify(window *Window) bool {
	var dummy _XEvent
	timeout := 0.1

	for !xCheckTypedWindowEvent(_glfw.platformWindow.display,
		window.platform.handle,
		_VisibilityNotify,
		&dummy) {
		if !waitForX11Event(&timeout) {
			return false
		}
	}

	return true
}

// getWindowState returns the state of the window.
func getWindowState(window *Window) int {
	result := _WithdrawnState

	var statePtr uintptr
	if getWindowPropertyX11(window.platform.handle,
		_glfw.platformWindow.WM_STATE,
		_glfw.platformWindow.WM_STATE,
		&statePtr) >= 2 {
		// The property contains a CARD32 state followed by the icon window,
		// each stored in a long.
		result = int(uint32(*(*_Culong)(unsafe.Pointer(statePtr))))
	}

	if statePtr != 0 {
		xFree(statePtr)
	}

	return result
}

// isSelectionEvent reports whether it is a selection event for the helper
// window.
func isSelectionEvent(display uintptr, eventPtr uintptr, pointer uintptr) uintptr {
	event := (*_XEvent)(unsafe.Pointer(eventPtr))
	if event.xany().Window != _glfw.platformWindow.helperWindowHandle {
		return 0
	}

	if event.EventType() == _SelectionRequest ||
		event.EventType() == _SelectionNotify ||
		event.EventType() == _SelectionClear {
		return 1
	}
	return 0
}

var isSelectionEventCallback uintptr

// isFrameExtentsEvent reports whether it is a _NET_FRAME_EXTENTS event for
// the window whose XID is passed as the pointer argument.
func isFrameExtentsEvent(display uintptr, eventPtr uintptr, pointer uintptr) uintptr {
	event := (*_XEvent)(unsafe.Pointer(eventPtr))
	if event.EventType() != _PropertyNotify {
		return 0
	}
	property := event.xproperty()
	if property.State == _PropertyNewValue &&
		property.Window == _XID(pointer) &&
		property.Atom == _glfw.platformWindow.NET_FRAME_EXTENTS {
		return 1
	}
	return 0
}

var isFrameExtentsEventCallback uintptr

// selPropNewValueNotification holds the SelectionNotify event whose property
// arrival the isSelPropNewValueNotify predicate matches. The predicate cannot
// receive the event through the XCheckIfEvent pointer argument, as a Go
// pointer must not be handed to C.
var selPropNewValueNotification *_XEvent

// isSelPropNewValueNotify reports whether it is a property event for the
// selection transfer awaited in selPropNewValueNotification.
func isSelPropNewValueNotify(display uintptr, eventPtr uintptr, pointer uintptr) uintptr {
	event := (*_XEvent)(unsafe.Pointer(eventPtr))
	if event.EventType() != _PropertyNotify {
		return 0
	}
	property := event.xproperty()
	notification := selPropNewValueNotification
	if property.State == _PropertyNewValue &&
		property.Window == notification.xselection().Requestor &&
		property.Atom == notification.xselection().Property {
		return 1
	}
	return 0
}

var isSelPropNewValueNotifyCallback uintptr

// translateState translates an X11 key state mask to GLFW modifier flags.
func translateState(state uint32) ModifierKey {
	var mods ModifierKey

	if state&_ShiftMask != 0 {
		mods |= ModShift
	}
	if state&_ControlMask != 0 {
		mods |= ModControl
	}
	if state&_Mod1Mask != 0 {
		mods |= ModAlt
	}
	if state&_Mod4Mask != 0 {
		mods |= ModSuper
	}
	if state&_LockMask != 0 {
		mods |= ModCapsLock
	}
	if state&_Mod2Mask != 0 {
		mods |= ModNumLock
	}

	return mods
}

// translateKey translates an X11 key code to a GLFW key token.
func translateKey(scancode int) Key {
	// Use the pre-filled LUT (see createKeyTables() in x11_init_linbsd.go)
	if scancode < 0 || scancode > 255 {
		return KeyUnknown
	}

	return _glfw.platformWindow.keycodes[scancode]
}

// sendEventToWM sends an EWMH or ICCCM event to the window manager.
func sendEventToWM(window *Window, eventType _Atom, a, b, c, d, e int) {
	var event _XEvent
	client := event.xclient()
	client.Type = _ClientMessage
	client.Window = window.platform.handle
	client.Format = 32 // Data is 32-bit longs
	client.MessageType = eventType
	client.Data[0] = _Clong(a)
	client.Data[1] = _Clong(b)
	client.Data[2] = _Clong(c)
	client.Data[3] = _Clong(d)
	client.Data[4] = _Clong(e)

	xSendEvent(_glfw.platformWindow.display, _glfw.platformWindow.root,
		false,
		_SubstructureNotifyMask|_SubstructureRedirectMask,
		&event)
}

func updateWindowHints(window *Window) {
	maximizable := false
	if window.monitor == nil && window.resizable && window.maxwidth == DontCare && window.maxheight == DontCare {
		maximizable = true
	}

	// flags, functions, decorations, input_mode, status
	var hints [5]_Culong

	hints[0] = _MWM_HINTS_FUNCTIONS | _MWM_HINTS_DECORATIONS

	hints[1] = _MWM_FUNC_MOVE | _MWM_FUNC_MINIMIZE | _MWM_FUNC_CLOSE
	if window.resizable && window.decorated {
		hints[1] |= _MWM_FUNC_RESIZE
	}
	if maximizable {
		hints[1] |= _MWM_FUNC_MAXIMIZE
	}

	if window.decorated {
		hints[2] |= _MWM_DECOR_BORDER | _MWM_DECOR_TITLE | _MWM_DECOR_MENU | _MWM_DECOR_MINIMIZE
		if window.resizable {
			hints[2] |= _MWM_DECOR_RESIZEH
		}
		if maximizable {
			hints[2] |= _MWM_DECOR_MAXIMIZE
		}
	}

	xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
		_glfw.platformWindow.MOTIF_WM_HINTS,
		_glfw.platformWindow.MOTIF_WM_HINTS, 32,
		_PropModeReplace,
		unsafe.Pointer(&hints[0]),
		int32(len(hints)))
}

// updateNormalHints updates the normal hints according to the window
// settings.
func updateNormalHints(window *Window, width, height int) {
	hintsPtr := xAllocSizeHints()
	hints := (*_XSizeHints)(unsafe.Pointer(hintsPtr))

	var supplied _Clong
	xGetWMNormalHints(_glfw.platformWindow.display, window.platform.handle, hints, &supplied)

	hints.Flags &^= _PMinSize | _PMaxSize | _PAspect

	if window.monitor == nil {
		if window.resizable {
			if window.minwidth != DontCare && window.minheight != DontCare {
				hints.Flags |= _PMinSize
				hints.MinWidth = int32(window.minwidth)
				hints.MinHeight = int32(window.minheight)
			}

			if window.maxwidth != DontCare && window.maxheight != DontCare {
				hints.Flags |= _PMaxSize
				hints.MaxWidth = int32(window.maxwidth)
				hints.MaxHeight = int32(window.maxheight)
			}

			if window.numer != DontCare && window.denom != DontCare {
				hints.Flags |= _PAspect
				hints.MinAspect.X = int32(window.numer)
				hints.MaxAspect.X = int32(window.numer)
				hints.MinAspect.Y = int32(window.denom)
				hints.MaxAspect.Y = int32(window.denom)
			}
		} else {
			hints.Flags |= _PMinSize | _PMaxSize
			hints.MinWidth = int32(width)
			hints.MaxWidth = int32(width)
			hints.MinHeight = int32(height)
			hints.MaxHeight = int32(height)
		}
	}

	xSetWMNormalHints(_glfw.platformWindow.display, window.platform.handle, hints)
	xFree(hintsPtr)

	updateWindowHints(window)
}

// updateWindowMode updates the full screen status of the window.
func updateWindowMode(window *Window) {
	if window.monitor != nil {
		if _glfw.platformWindow.xinerama.available &&
			_glfw.platformWindow.NET_WM_FULLSCREEN_MONITORS != 0 {
			sendEventToWM(window,
				_glfw.platformWindow.NET_WM_FULLSCREEN_MONITORS,
				window.monitor.platform.index,
				window.monitor.platform.index,
				window.monitor.platform.index,
				window.monitor.platform.index,
				0)
		}

		if _glfw.platformWindow.NET_WM_STATE != 0 && _glfw.platformWindow.NET_WM_STATE_FULLSCREEN != 0 {
			sendEventToWM(window,
				_glfw.platformWindow.NET_WM_STATE,
				_NET_WM_STATE_ADD,
				int(_glfw.platformWindow.NET_WM_STATE_FULLSCREEN),
				0, 1, 0)
		} else {
			// This is the butcher's way of removing window decorations
			// Setting the override-redirect attribute on a window makes the
			// window manager ignore the window completely (ICCCM, section 4)
			// The good thing is that this makes undecorated full screen windows
			// easy to do; the bad thing is that we have to do everything
			// manually and some things (like iconify/restore) won't work at
			// all, as those are tasks usually performed by the window manager

			var attributes _XSetWindowAttributes
			attributes.OverrideRedirect = 1
			xChangeWindowAttributes(_glfw.platformWindow.display,
				window.platform.handle,
				_CWOverrideRedirect,
				&attributes)

			window.platform.overrideRedirect = true
		}

		// Enable compositor bypass
		if !window.platform.transparent {
			value := _Culong(1)
			xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
				_glfw.platformWindow.NET_WM_BYPASS_COMPOSITOR, _XA_CARDINAL, 32,
				_PropModeReplace, unsafe.Pointer(&value), 1)
		}
	} else {
		if _glfw.platformWindow.xinerama.available &&
			_glfw.platformWindow.NET_WM_FULLSCREEN_MONITORS != 0 {
			xDeleteProperty(_glfw.platformWindow.display, window.platform.handle,
				_glfw.platformWindow.NET_WM_FULLSCREEN_MONITORS)
		}

		if _glfw.platformWindow.NET_WM_STATE != 0 && _glfw.platformWindow.NET_WM_STATE_FULLSCREEN != 0 {
			sendEventToWM(window,
				_glfw.platformWindow.NET_WM_STATE,
				_NET_WM_STATE_REMOVE,
				int(_glfw.platformWindow.NET_WM_STATE_FULLSCREEN),
				0, 1, 0)
		} else {
			var attributes _XSetWindowAttributes
			attributes.OverrideRedirect = 0
			xChangeWindowAttributes(_glfw.platformWindow.display,
				window.platform.handle,
				_CWOverrideRedirect,
				&attributes)

			window.platform.overrideRedirect = false
		}

		// Disable compositor bypass
		if !window.platform.transparent {
			xDeleteProperty(_glfw.platformWindow.display, window.platform.handle,
				_glfw.platformWindow.NET_WM_BYPASS_COMPOSITOR)
		}
	}
}

// convertLatin1toUTF8 converts the specified Latin-1 string to UTF-8.
func convertLatin1toUTF8(source string) string {
	codepoints := make([]rune, len(source))
	for i := 0; i < len(source); i++ {
		codepoints[i] = rune(source[i])
	}
	return string(codepoints)
}

// updateCursorImage updates the cursor image according to its cursor mode.
func updateCursorImage(window *Window) {
	if window.cursorMode == CursorNormal {
		if window.cursor != nil {
			xDefineCursor(_glfw.platformWindow.display, window.platform.handle,
				window.cursor.platform.handle)
		} else {
			xUndefineCursor(_glfw.platformWindow.display, window.platform.handle)
		}
	} else {
		xDefineCursor(_glfw.platformWindow.display, window.platform.handle,
			_glfw.platformWindow.hiddenCursorHandle)
	}
}

// captureCursor grabs the cursor and confines it to the window.
func captureCursor(window *Window) {
	xGrabPointer(_glfw.platformWindow.display, window.platform.handle, true,
		_ButtonPressMask|_ButtonReleaseMask|_PointerMotionMask,
		_GrabModeAsync, _GrabModeAsync,
		window.platform.handle,
		_None,
		_CurrentTime)
}

// releaseCursor ungrabs the cursor.
func releaseCursor() {
	xUngrabPointer(_glfw.platformWindow.display, _CurrentTime)
}

func xiMaskLen(event int) int {
	return (event >> 3) + 1
}

func xiSetMask(mask []byte, event int) {
	mask[event>>3] |= 1 << (uint(event) & 7)
}

func xiMaskIsSet(mask []byte, event int) bool {
	return mask[event>>3]&(1<<(uint(event)&7)) != 0
}

func enableRawMouseMotion(window *Window) {
	mask := make([]byte, xiMaskLen(_XI_RawMotion))
	xiSetMask(mask, _XI_RawMotion)

	em := _XIEventMask{
		Deviceid: _XIAllMasterDevices,
		MaskLen:  int32(len(mask)),
		Mask:     uintptr(unsafe.Pointer(&mask[0])),
	}
	_glfw.platformWindow.xi.SelectEvents(_glfw.platformWindow.display, _glfw.platformWindow.root, &em, 1)
	runtime.KeepAlive(mask)
}

func disableRawMouseMotion(window *Window) {
	mask := make([]byte, 1)

	em := _XIEventMask{
		Deviceid: _XIAllMasterDevices,
		MaskLen:  int32(len(mask)),
		Mask:     uintptr(unsafe.Pointer(&mask[0])),
	}
	_glfw.platformWindow.xi.SelectEvents(_glfw.platformWindow.display, _glfw.platformWindow.root, &em, 1)
	runtime.KeepAlive(mask)
}

// disableCursor applies disabled cursor mode to a focused window.
func disableCursor(window *Window) error {
	if window.rawMouseMotion {
		enableRawMouseMotion(window)
	}

	_glfw.platformWindow.disabledCursorWindow = window
	xpos, ypos, err := window.platformGetCursorPos()
	if err != nil {
		return err
	}
	_glfw.platformWindow.restoreCursorPosX = xpos
	_glfw.platformWindow.restoreCursorPosY = ypos
	updateCursorImage(window)
	if err := window.centerCursorInContentArea(); err != nil {
		return err
	}
	captureCursor(window)
	return nil
}

// enableCursor exits disabled cursor mode for the specified window.
func enableCursor(window *Window) error {
	if window.rawMouseMotion {
		disableRawMouseMotion(window)
	}

	_glfw.platformWindow.disabledCursorWindow = nil
	releaseCursor()
	if err := window.platformSetCursorPos(_glfw.platformWindow.restoreCursorPosX,
		_glfw.platformWindow.restoreCursorPosY); err != nil {
		return err
	}
	updateCursorImage(window)
	return nil
}

// createNativeWindow creates the X11 window.
func createNativeWindow(window *Window, wndconfig *wndconfig, visual uintptr, depth int32) error {
	width := wndconfig.width
	height := wndconfig.height

	if wndconfig.scaleToMonitor {
		width = int(float32(width) * _glfw.platformWindow.contentScaleX)
		height = int(float32(height) * _glfw.platformWindow.contentScaleY)
	}

	// Create a colormap based on the visual used by the current context
	window.platform.colormap = xCreateColormap(_glfw.platformWindow.display,
		_glfw.platformWindow.root,
		visual,
		_AllocNone)

	window.platform.transparent = isVisualTransparentX11(visual)

	var wa _XSetWindowAttributes
	wa.Colormap = window.platform.colormap
	wa.EventMask = _StructureNotifyMask | _KeyPressMask | _KeyReleaseMask |
		_PointerMotionMask | _ButtonPressMask | _ButtonReleaseMask |
		_ExposureMask | _FocusChangeMask | _VisibilityChangeMask |
		_EnterWindowMask | _LeaveWindowMask | _PropertyChangeMask

	grabErrorHandlerX11()

	window.platform.parent = _glfw.platformWindow.root
	window.platform.handle = xCreateWindow(_glfw.platformWindow.display,
		_glfw.platformWindow.root,
		0, 0, // Position
		uint32(width), uint32(height),
		0,     // Border width
		depth, // Color depth
		_InputOutput,
		visual,
		_CWBorderPixel|_CWColormap|_CWEventMask,
		&wa)

	releaseErrorHandlerX11()

	if window.platform.handle == 0 {
		return inputErrorX11(PlatformError, "X11: Failed to create window")
	}

	_glfw.platformWindow.windowsByXID[window.platform.handle] = window

	if !wndconfig.decorated {
		if err := window.platformSetWindowDecorated(false); err != nil {
			return err
		}
	}

	if _glfw.platformWindow.NET_WM_STATE != 0 && window.monitor == nil {
		states := make([]_Atom, 0, 3)

		if wndconfig.floating {
			if _glfw.platformWindow.NET_WM_STATE_ABOVE != 0 {
				states = append(states, _glfw.platformWindow.NET_WM_STATE_ABOVE)
			}
		}

		if wndconfig.maximized {
			if _glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT != 0 &&
				_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ != 0 {
				states = append(states,
					_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT,
					_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ)
				window.platform.maximized = true
			}
		}

		if len(states) > 0 {
			xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
				_glfw.platformWindow.NET_WM_STATE, _XA_ATOM, 32,
				_PropModeReplace, unsafe.Pointer(&states[0]), int32(len(states)))
		}
	}

	// Whether _NET_WM_SYNC_REQUEST frame synchronization can be used for this
	// window. It requires the X Sync extension and both EWMH atoms.
	syncRequestSupported := _glfw.platformWindow.xsync.available &&
		_glfw.platformWindow.NET_WM_SYNC_REQUEST != 0 &&
		_glfw.platformWindow.NET_WM_SYNC_REQUEST_COUNTER != 0

	// Declare the WM protocols supported by GLFW
	{
		protocols := []_Atom{
			_glfw.platformWindow.WM_DELETE_WINDOW,
			_glfw.platformWindow.NET_WM_PING,
		}

		// Advertise _NET_WM_SYNC_REQUEST only when the counter below can be set;
		// otherwise a compositor would wait on it forever during a resize.
		if syncRequestSupported {
			protocols = append(protocols, _glfw.platformWindow.NET_WM_SYNC_REQUEST)
		}

		xSetWMProtocols(_glfw.platformWindow.display, window.platform.handle,
			&protocols[0], int32(len(protocols)))
	}

	// Create the _NET_WM_SYNC_REQUEST counter. The window manager reads it to
	// tell when a frame has been drawn at the new size while resizing, so it can
	// hold the previous frame instead of showing the unpainted background.
	if syncRequestSupported {
		window.platform.syncCounter = _glfw.platformWindow.xsync.CreateCounter(
			_glfw.platformWindow.display, 0)

		counter := _Clong(window.platform.syncCounter)
		xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
			_glfw.platformWindow.NET_WM_SYNC_REQUEST_COUNTER, _XA_CARDINAL, 32,
			_PropModeReplace,
			unsafe.Pointer(&counter), 1)
	}

	// Declare our PID
	{
		pid := _Clong(os.Getpid())
		xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
			_glfw.platformWindow.NET_WM_PID, _XA_CARDINAL, 32,
			_PropModeReplace,
			unsafe.Pointer(&pid), 1)
	}

	if _glfw.platformWindow.NET_WM_WINDOW_TYPE != 0 && _glfw.platformWindow.NET_WM_WINDOW_TYPE_NORMAL != 0 {
		windowType := _glfw.platformWindow.NET_WM_WINDOW_TYPE_NORMAL
		xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
			_glfw.platformWindow.NET_WM_WINDOW_TYPE, _XA_ATOM, 32,
			_PropModeReplace, unsafe.Pointer(&windowType), 1)
	}

	// Set ICCCM WM_HINTS property
	{
		hintsPtr := xAllocWMHints()
		if hintsPtr == 0 {
			return fmt.Errorf("glfw: x11: failed to allocate WM hints: %w", OutOfMemory)
		}
		hints := (*_XWMHints)(unsafe.Pointer(hintsPtr))

		hints.Flags = _StateHint
		hints.InitialState = _NormalState

		xSetWMHints(_glfw.platformWindow.display, window.platform.handle, hints)
		xFree(hintsPtr)
	}

	// Set ICCCM WM_NORMAL_HINTS property
	{
		hintsPtr := xAllocSizeHints()
		if hintsPtr == 0 {
			return fmt.Errorf("glfw: x11: failed to allocate size hints: %w", OutOfMemory)
		}
		hints := (*_XSizeHints)(unsafe.Pointer(hintsPtr))

		if !wndconfig.resizable {
			hints.Flags |= _PMinSize | _PMaxSize
			hints.MinWidth = int32(width)
			hints.MaxWidth = int32(width)
			hints.MinHeight = int32(height)
			hints.MaxHeight = int32(height)
		}

		hints.Flags |= _PWinGravity
		hints.WinGravity = _StaticGravity

		xSetWMNormalHints(_glfw.platformWindow.display, window.platform.handle, hints)
		xFree(hintsPtr)
	}

	// Set ICCCM WM_CLASS property
	{
		var name, class string

		if len(wndconfig.instanceName) > 0 && len(wndconfig.className) > 0 {
			name = wndconfig.instanceName
			class = wndconfig.className
		} else {
			resourceName := os.Getenv("RESOURCE_NAME")
			if len(resourceName) > 0 {
				name = resourceName
			} else if len(wndconfig.title) > 0 {
				name = wndconfig.title
			} else {
				name = "glfw-application"
			}

			if len(wndconfig.title) > 0 {
				class = wndconfig.title
			} else {
				class = "GLFW-Application"
			}
		}

		nameBytes := append([]byte(name), 0)
		classBytes := append([]byte(class), 0)
		hint := _XClassHint{
			ResName:  uintptr(unsafe.Pointer(&nameBytes[0])),
			ResClass: uintptr(unsafe.Pointer(&classBytes[0])),
		}

		xSetClassHint(_glfw.platformWindow.display, window.platform.handle, &hint)
		runtime.KeepAlive(nameBytes)
		runtime.KeepAlive(classBytes)
	}

	// Announce support for Xdnd (drag and drop)
	{
		version := _Atom(_GLFW_XDND_VERSION)
		xChangeProperty(_glfw.platformWindow.display, window.platform.handle,
			_glfw.platformWindow.XdndAware, _XA_ATOM, 32,
			_PropModeReplace, unsafe.Pointer(&version), 1)
	}

	if err := window.platformSetWindowTitle(wndconfig.title); err != nil {
		return err
	}

	if _glfw.platformWindow.im != 0 {
		window.platform.ic = xCreateIC(_glfw.platformWindow.im,
			"inputStyle",
			_XIMPreeditNothing|_XIMStatusNothing,
			"clientWindow",
			window.platform.handle,
			"focusWindow",
			window.platform.handle,
			0)
	}

	if window.platform.ic != 0 {
		var filter _Culong
		if xGetICValues(window.platform.ic, "filterEvents", &filter, 0) == 0 {
			xSelectInput(_glfw.platformWindow.display, window.platform.handle, wa.EventMask|_Clong(filter))
		}
	}

	var err error
	window.platform.xpos, window.platform.ypos, err = window.platformGetWindowPos()
	if err != nil {
		return err
	}
	window.platform.width, window.platform.height, err = window.platformGetWindowSize()
	if err != nil {
		return err
	}

	return nil
}

// writeTargetToProperty writes the selection string in the format of the
// requested target and returns the property to reply with, or None when the
// request cannot be satisfied.
func writeTargetToProperty(request *_XSelectionRequestEvent) _Atom {
	formats := []_Atom{_glfw.platformWindow.UTF8_STRING, _XA_STRING}

	selectionString := _glfw.platformWindow.clipboardString
	if request.Selection == _glfw.platformWindow.PRIMARY {
		selectionString = _glfw.platformWindow.primarySelectionString
	}
	selectionBytes := []byte(selectionString)
	var selectionPtr unsafe.Pointer
	if len(selectionBytes) > 0 {
		selectionPtr = unsafe.Pointer(&selectionBytes[0])
	}

	if request.Property == _None {
		// The requester is a legacy client (ICCCM section 2.2)
		// Legacy clients are not supported, so fail here
		return _None
	}

	if request.Target == _glfw.platformWindow.TARGETS {
		// The list of supported targets was requested

		targets := []_Atom{
			_glfw.platformWindow.TARGETS,
			_glfw.platformWindow.MULTIPLE,
			_glfw.platformWindow.UTF8_STRING,
			_XA_STRING,
		}

		xChangeProperty(_glfw.platformWindow.display,
			request.Requestor,
			request.Property,
			_XA_ATOM,
			32,
			_PropModeReplace,
			unsafe.Pointer(&targets[0]),
			int32(len(targets)))

		return request.Property
	}

	if request.Target == _glfw.platformWindow.MULTIPLE {
		// Multiple conversions were requested

		var targetsPtr uintptr
		count := getWindowPropertyX11(request.Requestor,
			request.Property,
			_glfw.platformWindow.ATOM_PAIR,
			&targetsPtr)

		var targets []_Atom
		if targetsPtr != 0 && count > 0 {
			targets = unsafe.Slice((*_Atom)(unsafe.Pointer(targetsPtr)), count)
		}

		for i := 0; i+1 < len(targets); i += 2 {
			supported := false
			for _, format := range formats {
				if targets[i] == format {
					supported = true
					break
				}
			}

			if supported {
				xChangeProperty(_glfw.platformWindow.display,
					request.Requestor,
					targets[i+1],
					targets[i],
					8,
					_PropModeReplace,
					selectionPtr,
					int32(len(selectionBytes)))
			} else {
				targets[i+1] = _None
			}
		}

		xChangeProperty(_glfw.platformWindow.display,
			request.Requestor,
			request.Property,
			_glfw.platformWindow.ATOM_PAIR,
			32,
			_PropModeReplace,
			unsafe.Pointer(targetsPtr),
			int32(count))

		if targetsPtr != 0 {
			xFree(targetsPtr)
		}

		return request.Property
	}

	if request.Target == _glfw.platformWindow.SAVE_TARGETS {
		// The request is a check whether SAVE_TARGETS is supported
		// It should be handled as a no-op side effect target

		xChangeProperty(_glfw.platformWindow.display,
			request.Requestor,
			request.Property,
			_glfw.platformWindow.NULL_,
			32,
			_PropModeReplace,
			nil,
			0)

		return request.Property
	}

	// Conversion to a data target was requested

	for _, format := range formats {
		if request.Target == format {
			// The requested target is one that is supported

			xChangeProperty(_glfw.platformWindow.display,
				request.Requestor,
				request.Property,
				request.Target,
				8,
				_PropModeReplace,
				selectionPtr,
				int32(len(selectionBytes)))

			return request.Property
		}
	}

	// The requested target is not supported

	return _None
}

// handleSelectionRequest replies to a request to convert the selection.
func handleSelectionRequest(event *_XEvent) {
	request := event.xselectionrequest()

	var reply _XEvent
	selection := reply.xselection()
	selection.Type = _SelectionNotify
	selection.Property = writeTargetToProperty(request)
	selection.Display = request.Display
	selection.Requestor = request.Requestor
	selection.Selection = request.Selection
	selection.Target = request.Target
	selection.Time = request.Time

	xSendEvent(_glfw.platformWindow.display, request.Requestor, false, 0, &reply)
}

// getSelectionString returns the string held by the specified selection.
func getSelectionString(selection _Atom) (string, error) {
	selectionString := &_glfw.platformWindow.clipboardString
	if selection == _glfw.platformWindow.PRIMARY {
		selectionString = &_glfw.platformWindow.primarySelectionString
	}

	if xGetSelectionOwner(_glfw.platformWindow.display, selection) ==
		_glfw.platformWindow.helperWindowHandle {
		// Instead of doing a large number of X round-trips just to put this
		// string into a window property and then read it back, just return it
		return *selectionString, nil
	}

	*selectionString = ""

	if isSelPropNewValueNotifyCallback == 0 {
		isSelPropNewValueNotifyCallback = purego.NewCallback(isSelPropNewValueNotify)
	}

	found := false
	for _, target := range []_Atom{_glfw.platformWindow.UTF8_STRING, _XA_STRING} {
		var notification, dummy _XEvent

		xConvertSelection(_glfw.platformWindow.display,
			selection,
			target,
			_glfw.platformWindow.GLFW_SELECTION,
			_glfw.platformWindow.helperWindowHandle,
			_CurrentTime)

		for !xCheckTypedWindowEvent(_glfw.platformWindow.display,
			_glfw.platformWindow.helperWindowHandle,
			_SelectionNotify,
			&notification) {
			waitForX11Event(nil)
		}

		if notification.xselection().Property == _None {
			continue
		}

		selPropNewValueNotification = &notification
		xCheckIfEvent(_glfw.platformWindow.display,
			&dummy,
			isSelPropNewValueNotifyCallback,
			0)

		var data uintptr
		var actualType _Atom
		var actualFormat int32
		var itemCount, bytesAfter _Culong

		xGetWindowProperty(_glfw.platformWindow.display,
			notification.xselection().Requestor,
			notification.xselection().Property,
			0,
			math.MaxInt,
			true,
			_AnyPropertyType,
			&actualType,
			&actualFormat,
			&itemCount,
			&bytesAfter,
			&data)

		if actualType == _glfw.platformWindow.INCR {
			var str []byte
			received := false

			for {
				for !xCheckIfEvent(_glfw.platformWindow.display,
					&dummy,
					isSelPropNewValueNotifyCallback,
					0) {
					waitForX11Event(nil)
				}

				if data != 0 {
					xFree(data)
				}
				xGetWindowProperty(_glfw.platformWindow.display,
					notification.xselection().Requestor,
					notification.xselection().Property,
					0,
					math.MaxInt,
					true,
					_AnyPropertyType,
					&actualType,
					&actualFormat,
					&itemCount,
					&bytesAfter,
					&data)

				if itemCount != 0 {
					received = true
					str = append(str, goString(data)...)
				}

				if itemCount == 0 {
					if received {
						if target == _XA_STRING {
							*selectionString = convertLatin1toUTF8(string(str))
						} else {
							*selectionString = string(str)
						}
						found = true
					}

					break
				}
			}
		} else if actualType == target {
			if target == _XA_STRING {
				*selectionString = convertLatin1toUTF8(goString(data))
			} else {
				*selectionString = goString(data)
			}
			found = true
		}

		if data != 0 {
			xFree(data)
		}

		if found {
			break
		}
	}

	if !found {
		return "", fmt.Errorf("glfw: x11: failed to convert selection to string: %w", FormatUnavailable)
	}

	return *selectionString, nil
}

// acquireMonitor makes the window and its video mode active on its monitor.
func acquireMonitor(window *Window) error {
	if _glfw.platformWindow.saver.count == 0 {
		// Remember old screen saver settings
		xGetScreenSaver(_glfw.platformWindow.display,
			&_glfw.platformWindow.saver.timeout,
			&_glfw.platformWindow.saver.interval,
			&_glfw.platformWindow.saver.blanking,
			&_glfw.platformWindow.saver.exposure)

		// Disable screen saver
		xSetScreenSaver(_glfw.platformWindow.display, 0, 0, _DontPreferBlanking, _DefaultExposures)
	}

	if window.monitor.window == nil {
		_glfw.platformWindow.saver.count++
	}

	if err := setVideoModeX11(window.monitor, &window.videoMode); err != nil {
		return err
	}

	if window.platform.overrideRedirect {
		// Manually position the window over its monitor
		xpos, ypos, _ := window.monitor.platformGetMonitorPos()
		mode := window.monitor.platformGetVideoMode()

		xMoveResizeWindow(_glfw.platformWindow.display, window.platform.handle,
			int32(xpos), int32(ypos), uint32(mode.Width), uint32(mode.Height))
	}

	window.monitor.inputMonitorWindow(window)
	return nil
}

// releaseMonitor removes the window and restores the original video mode.
func releaseMonitor(window *Window) {
	if window.monitor.window != window {
		return
	}

	window.monitor.inputMonitorWindow(nil)
	restoreVideoModeX11(window.monitor)

	_glfw.platformWindow.saver.count--

	if _glfw.platformWindow.saver.count == 0 {
		// Restore old screen saver settings
		xSetScreenSaver(_glfw.platformWindow.display,
			_glfw.platformWindow.saver.timeout,
			_glfw.platformWindow.saver.interval,
			_glfw.platformWindow.saver.blanking,
			_glfw.platformWindow.saver.exposure)
	}
}

// processEvent processes the specified X event.
func processEvent(event *_XEvent) error {
	var keycode int
	filtered := false

	// HACK: Save scancode as some IMs clear the field in XFilterEvent
	if event.EventType() == _KeyPress || event.EventType() == _KeyRelease {
		keycode = int(event.xkey().Keycode)
	}

	if _glfw.platformWindow.im != 0 {
		filtered = xFilterEvent(event, _None)
	}

	if _glfw.platformWindow.randr.available {
		if event.EventType() == _glfw.platformWindow.randr.eventBase+_RRNotify {
			_glfw.platformWindow.randr.UpdateConfiguration(event)
			return pollMonitorsX11()
		}
	}

	if _glfw.platformWindow.xkb.available && event.EventType() == _glfw.platformWindow.xkb.eventBase+_XkbEventCode {
		switch event.xkbAny().XkbType {
		case _XkbStateNotify:
			if event.xkbState().Changed&_XkbGroupStateMask != 0 {
				_glfw.platformWindow.xkb.group = uint32(event.xkbState().Group)
			}
		case _XkbMapNotify:
			createKeyTables()
		}

		return nil
	}

	if event.EventType() == _GenericEvent {
		if _glfw.platformWindow.xi.available {
			window := _glfw.platformWindow.disabledCursorWindow
			cookie := event.xcookie()

			if window != nil &&
				window.rawMouseMotion &&
				cookie.Extension == _glfw.platformWindow.xi.majorOpcode &&
				xGetEventData(_glfw.platformWindow.display, cookie) &&
				cookie.Evtype == _XI_RawMotion {
				re := (*_XIRawEvent)(unsafe.Pointer(cookie.Data))
				if re.Valuators.MaskLen != 0 {
					mask := unsafe.Slice((*byte)(unsafe.Pointer(re.Valuators.Mask)), re.Valuators.MaskLen)
					values := re.RawValues
					xpos := window.virtualCursorPosX
					ypos := window.virtualCursorPosY

					if xiMaskIsSet(mask, 0) {
						xpos += *(*float64)(unsafe.Pointer(values))
						values += unsafe.Sizeof(float64(0))
					}

					if xiMaskIsSet(mask, 1) {
						ypos += *(*float64)(unsafe.Pointer(values))
					}

					window.inputCursorPos(xpos, ypos)
				}
			}

			xFreeEventData(_glfw.platformWindow.display, cookie)
		}

		return nil
	}

	if event.EventType() == _SelectionRequest {
		handleSelectionRequest(event)
		return nil
	}

	window := _glfw.platformWindow.windowsByXID[event.xany().Window]
	if window == nil {
		// This is an event for a window that has already been destroyed
		return nil
	}

	switch event.EventType() {
	case _ReparentNotify:
		window.platform.parent = event.xreparent().Parent
		return nil

	case _KeyPress:
		key := translateKey(keycode)
		mods := translateState(event.xkey().State)
		plain := mods&(ModControl|ModAlt) == 0

		if window.platform.ic != 0 {
			// HACK: Do not report the key press events duplicated by XIM
			//       Duplicate key releases are filtered out implicitly by
			//       the GLFW key repeat logic in inputKey
			//       A timestamp per key is used to handle simultaneous keys
			// NOTE: Always allow the first event for each key through
			//       (the server never sends a timestamp of zero)
			// NOTE: Timestamp difference is compared to handle wrap-around
			diff := event.xkey().Time - window.platform.keyPressTimes[keycode]
			if diff == event.xkey().Time || (diff > 0 && diff < 1<<31) {
				if keycode != 0 {
					window.inputKey(key, keycode, Press, mods)
				}

				window.platform.keyPressTimes[keycode] = event.xkey().Time
			}

			if !filtered {
				var status int32
				buffer := make([]byte, 100)
				chars := buffer

				count := xutf8LookupString(window.platform.ic,
					event.xkey(),
					chars, int32(len(buffer)-1),
					nil, &status)

				if status == _XBufferOverflow {
					chars = make([]byte, count+1)
					count = xutf8LookupString(window.platform.ic,
						event.xkey(),
						chars, count,
						nil, &status)
				}

				if status == _XLookupChars || status == _XLookupBoth {
					for _, codepoint := range string(chars[:count]) {
						window.inputChar(codepoint, mods, plain)
					}
				}
			}
		} else {
			var keysym _KeySym
			xLookupString(event.xkey(), nil, 0, &keysym, 0)

			window.inputKey(key, keycode, Press, mods)

			if codepoint, ok := keySym2Unicode(uint32(keysym)); ok {
				window.inputChar(codepoint, mods, plain)
			}
		}

		return nil

	case _KeyRelease:
		key := translateKey(keycode)
		mods := translateState(event.xkey().State)

		if !_glfw.platformWindow.xkb.detectable {
			// HACK: Key repeat events will arrive as KeyRelease/KeyPress
			//       pairs with similar or identical time stamps
			//       The key repeat logic in inputKey expects only key
			//       presses to repeat, so detect and discard release events
			if xEventsQueued(_glfw.platformWindow.display, _QueuedAfterReading) != 0 {
				var next _XEvent
				xPeekEvent(_glfw.platformWindow.display, &next)

				if next.EventType() == _KeyPress &&
					next.xkey().Window == event.xkey().Window &&
					int(next.xkey().Keycode) == keycode {
					// HACK: The time of repeat events sometimes doesn't
					//       match that of the press event, so add an
					//       epsilon
					//       Toshiyuki Takahashi can press a button
					//       16 times per second so it's fairly safe to
					//       assume that no human is pressing the key 50
					//       times per second (value is ms)
					if next.xkey().Time-event.xkey().Time < 20 {
						// This is very likely a server-generated key repeat
						// event, so ignore it
						return nil
					}
				}
			}
		}

		window.inputKey(key, keycode, Release, mods)
		return nil

	case _ButtonPress:
		mods := translateState(event.xbutton().State)

		switch event.xbutton().Button {
		case _Button1:
			window.inputMouseClick(MouseButtonLeft, Press, mods)
		case _Button2:
			window.inputMouseClick(MouseButtonMiddle, Press, mods)
		case _Button3:
			window.inputMouseClick(MouseButtonRight, Press, mods)

		// Modern X provides scroll events as mouse button presses
		case _Button4:
			window.inputScroll(0, 1)
		case _Button5:
			window.inputScroll(0, -1)
		case _Button6:
			window.inputScroll(1, 0)
		case _Button7:
			window.inputScroll(-1, 0)

		default:
			// Additional buttons after 7 are treated as regular buttons
			// The gap left by the scroll input above is filled by
			// subtracting 4
			window.inputMouseClick(MouseButton(int(event.xbutton().Button)-_Button1-4),
				Press, mods)
		}

		return nil

	case _ButtonRelease:
		mods := translateState(event.xbutton().State)

		switch {
		case event.xbutton().Button == _Button1:
			window.inputMouseClick(MouseButtonLeft, Release, mods)
		case event.xbutton().Button == _Button2:
			window.inputMouseClick(MouseButtonMiddle, Release, mods)
		case event.xbutton().Button == _Button3:
			window.inputMouseClick(MouseButtonRight, Release, mods)
		case event.xbutton().Button > _Button7:
			// Additional buttons after 7 are treated as regular buttons
			// The gap left by the scroll input above is filled by
			// subtracting 4
			window.inputMouseClick(MouseButton(int(event.xbutton().Button)-_Button1-4),
				Release, mods)
		}

		return nil

	case _EnterNotify:
		// XEnterWindowEvent is XCrossingEvent
		x := int(event.xcrossing().X)
		y := int(event.xcrossing().Y)

		// HACK: This is a workaround for WMs (KWM, Fluxbox) that otherwise
		//       ignore the defined cursor for hidden cursor mode
		if window.cursorMode == CursorHidden {
			updateCursorImage(window)
		}

		window.inputCursorEnter(true)
		window.inputCursorPos(float64(x), float64(y))

		window.platform.lastCursorPosX = x
		window.platform.lastCursorPosY = y
		return nil

	case _LeaveNotify:
		window.inputCursorEnter(false)
		return nil

	case _MotionNotify:
		x := int(event.xmotion().X)
		y := int(event.xmotion().Y)

		if x != window.platform.warpCursorPosX ||
			y != window.platform.warpCursorPosY {
			// The cursor was moved by something other than GLFW

			if window.cursorMode == CursorDisabled {
				if _glfw.platformWindow.disabledCursorWindow != window {
					return nil
				}
				if window.rawMouseMotion {
					return nil
				}

				dx := x - window.platform.lastCursorPosX
				dy := y - window.platform.lastCursorPosY

				window.inputCursorPos(window.virtualCursorPosX+float64(dx),
					window.virtualCursorPosY+float64(dy))
			} else {
				window.inputCursorPos(float64(x), float64(y))
			}
		}

		window.platform.lastCursorPosX = x
		window.platform.lastCursorPosY = y
		return nil

	case _ConfigureNotify:
		if int(event.xconfigure().Width) != window.platform.width ||
			int(event.xconfigure().Height) != window.platform.height {
			window.platform.width = int(event.xconfigure().Width)
			window.platform.height = int(event.xconfigure().Height)

			window.inputFramebufferSize(int(event.xconfigure().Width),
				int(event.xconfigure().Height))

			window.inputWindowSize(int(event.xconfigure().Width),
				int(event.xconfigure().Height))
		}

		xpos := event.xconfigure().X
		ypos := event.xconfigure().Y

		// NOTE: ConfigureNotify events from the server are in local
		//       coordinates, so the position of a reparented window needs
		//       to be translated into root (screen) coordinates
		if event.xany().SendEvent == 0 && window.platform.parent != _glfw.platformWindow.root {
			grabErrorHandlerX11()

			var dummy _XID
			xTranslateCoordinates(_glfw.platformWindow.display,
				window.platform.parent,
				_glfw.platformWindow.root,
				xpos, ypos,
				&xpos, &ypos,
				&dummy)

			releaseErrorHandlerX11()
			if _glfw.platformWindow.errorCode == _BadWindow {
				return nil
			}
		}

		if int(xpos) != window.platform.xpos || int(ypos) != window.platform.ypos {
			window.platform.xpos = int(xpos)
			window.platform.ypos = int(ypos)

			window.inputWindowPos(int(xpos), int(ypos))
		}

		return nil

	case _ClientMessage:
		// Custom client message, probably from the window manager

		if filtered {
			return nil
		}

		client := event.xclient()

		if client.MessageType == _None {
			return nil
		}

		if client.MessageType == _glfw.platformWindow.WM_PROTOCOLS {
			protocol := _Atom(client.Data[0])
			if protocol == _None {
				return nil
			}

			if protocol == _glfw.platformWindow.WM_DELETE_WINDOW {
				// The window manager was asked to close the window, for
				// example by the user pressing a 'close' window decoration
				// button
				window.inputWindowCloseRequest()
			} else if protocol == _glfw.platformWindow.NET_WM_PING {
				// The window manager is pinging the application to ensure
				// it's still responding to events

				reply := *event
				reply.xclient().Window = _glfw.platformWindow.root

				xSendEvent(_glfw.platformWindow.display, _glfw.platformWindow.root,
					false,
					_SubstructureNotifyMask|_SubstructureRedirectMask,
					&reply)
			} else if protocol == _glfw.platformWindow.NET_WM_SYNC_REQUEST {
				// The window manager is (interactively) resizing the window and
				// wants the sync counter set to this value once a frame has been
				// drawn at the new size (EWMH _NET_WM_SYNC_REQUEST). Data[2] and
				// Data[3] carry the low and high halves of the requested value.
				if window.platform.syncCounter != 0 {
					packed := uint64(uint32(int32(client.Data[3])))<<32 | uint64(uint32(client.Data[2]))
					window.platform.syncValue.Store(packed)
					window.platform.syncRequested.Store(true)
				}
			}
		} else if client.MessageType == _glfw.platformWindow.XdndEnter {
			// A drag operation has entered the window
			list := client.Data[1]&1 != 0

			_glfw.platformWindow.xdnd.source = _XID(client.Data[0])
			_glfw.platformWindow.xdnd.version = client.Data[1] >> 24
			_glfw.platformWindow.xdnd.format = _None

			if _glfw.platformWindow.xdnd.version > _GLFW_XDND_VERSION {
				return nil
			}

			var formats []_Atom
			var formatsPtr uintptr

			if list {
				count := getWindowPropertyX11(_glfw.platformWindow.xdnd.source,
					_glfw.platformWindow.XdndTypeList,
					_XA_ATOM,
					&formatsPtr)
				if formatsPtr != 0 && count > 0 {
					formats = unsafe.Slice((*_Atom)(unsafe.Pointer(formatsPtr)), count)
				}
			} else {
				formats = []_Atom{_Atom(client.Data[2]), _Atom(client.Data[3]), _Atom(client.Data[4])}
			}

			for _, format := range formats {
				if format == _glfw.platformWindow.text_uri_list {
					_glfw.platformWindow.xdnd.format = _glfw.platformWindow.text_uri_list
					break
				}
			}

			if formatsPtr != 0 {
				xFree(formatsPtr)
			}
		} else if client.MessageType == _glfw.platformWindow.XdndDrop {
			// The drag operation has finished by dropping on the window
			time := _CurrentTime

			if _glfw.platformWindow.xdnd.version > _GLFW_XDND_VERSION {
				return nil
			}

			if _glfw.platformWindow.xdnd.format != _None {
				if _glfw.platformWindow.xdnd.version >= 1 {
					time = _Time(client.Data[2])
				}

				// Request the chosen format from the source window
				xConvertSelection(_glfw.platformWindow.display,
					_glfw.platformWindow.XdndSelection,
					_glfw.platformWindow.xdnd.format,
					_glfw.platformWindow.XdndSelection,
					window.platform.handle,
					time)
			} else if _glfw.platformWindow.xdnd.version >= 2 {
				var reply _XEvent
				replyClient := reply.xclient()
				replyClient.Type = _ClientMessage
				replyClient.Window = _glfw.platformWindow.xdnd.source
				replyClient.MessageType = _glfw.platformWindow.XdndFinished
				replyClient.Format = 32
				replyClient.Data[0] = _Clong(window.platform.handle)
				replyClient.Data[1] = 0 // The drag was rejected
				replyClient.Data[2] = _None

				xSendEvent(_glfw.platformWindow.display, _glfw.platformWindow.xdnd.source,
					false, _NoEventMask, &reply)
				xFlush(_glfw.platformWindow.display)
			}
		} else if client.MessageType == _glfw.platformWindow.XdndPosition {
			// The drag operation has moved over the window
			xabs := int32((client.Data[2] >> 16) & 0xffff)
			yabs := int32(client.Data[2] & 0xffff)

			if _glfw.platformWindow.xdnd.version > _GLFW_XDND_VERSION {
				return nil
			}

			var dummy _XID
			var xpos, ypos int32
			xTranslateCoordinates(_glfw.platformWindow.display,
				_glfw.platformWindow.root,
				window.platform.handle,
				xabs, yabs,
				&xpos, &ypos,
				&dummy)

			window.inputCursorPos(float64(xpos), float64(ypos))

			var reply _XEvent
			replyClient := reply.xclient()
			replyClient.Type = _ClientMessage
			replyClient.Window = _glfw.platformWindow.xdnd.source
			replyClient.MessageType = _glfw.platformWindow.XdndStatus
			replyClient.Format = 32
			replyClient.Data[0] = _Clong(window.platform.handle)
			replyClient.Data[2] = 0 // Specify an empty rectangle
			replyClient.Data[3] = 0

			if _glfw.platformWindow.xdnd.format != _None {
				// Reply that we are ready to copy the dragged data
				replyClient.Data[1] = 1 // Accept with no rectangle
				if _glfw.platformWindow.xdnd.version >= 2 {
					replyClient.Data[4] = _Clong(_glfw.platformWindow.XdndActionCopy)
				}
			}

			xSendEvent(_glfw.platformWindow.display, _glfw.platformWindow.xdnd.source,
				false, _NoEventMask, &reply)
			xFlush(_glfw.platformWindow.display)
		}

		return nil

	case _SelectionNotify:
		if event.xselection().Property == _glfw.platformWindow.XdndSelection {
			// The converted data from the drag operation has arrived
			var data uintptr
			result := getWindowPropertyX11(event.xselection().Requestor,
				event.xselection().Property,
				event.xselection().Target,
				&data)

			if result != 0 {
				window.inputDrop(parseUriList(goString(data)))
			}

			if data != 0 {
				xFree(data)
			}

			if _glfw.platformWindow.xdnd.version >= 2 {
				var reply _XEvent
				replyClient := reply.xclient()
				replyClient.Type = _ClientMessage
				replyClient.Window = _glfw.platformWindow.xdnd.source
				replyClient.MessageType = _glfw.platformWindow.XdndFinished
				replyClient.Format = 32
				replyClient.Data[0] = _Clong(window.platform.handle)
				replyClient.Data[1] = _Clong(result)
				replyClient.Data[2] = _Clong(_glfw.platformWindow.XdndActionCopy)

				xSendEvent(_glfw.platformWindow.display, _glfw.platformWindow.xdnd.source,
					false, _NoEventMask, &reply)
				xFlush(_glfw.platformWindow.display)
			}
		}

		return nil

	case _FocusIn:
		if event.xfocus().Mode == _NotifyGrab ||
			event.xfocus().Mode == _NotifyUngrab {
			// Ignore focus events from popup indicator windows, window menu
			// key chords and window dragging
			return nil
		}

		if window.cursorMode == CursorDisabled {
			if err := disableCursor(window); err != nil {
				return err
			}
		}

		if window.platform.ic != 0 {
			xSetICFocus(window.platform.ic)
		}

		window.inputWindowFocus(true)
		return nil

	case _FocusOut:
		if event.xfocus().Mode == _NotifyGrab ||
			event.xfocus().Mode == _NotifyUngrab {
			// Ignore focus events from popup indicator windows, window menu
			// key chords and window dragging
			return nil
		}

		if window.cursorMode == CursorDisabled {
			if err := enableCursor(window); err != nil {
				return err
			}
		}

		if window.platform.ic != 0 {
			xUnsetICFocus(window.platform.ic)
		}

		if window.monitor != nil && window.autoIconify {
			window.platformIconifyWindow()
		}

		window.inputWindowFocus(false)
		return nil

	case _Expose:
		window.inputWindowDamage()
		return nil

	case _PropertyNotify:
		if event.xproperty().State != _PropertyNewValue {
			return nil
		}

		if event.xproperty().Atom == _glfw.platformWindow.WM_STATE {
			state := getWindowState(window)
			if state != _IconicState && state != _NormalState {
				return nil
			}

			iconified := state == _IconicState
			if window.platform.iconified != iconified {
				if window.monitor != nil {
					if iconified {
						releaseMonitor(window)
					} else {
						if err := acquireMonitor(window); err != nil {
							return err
						}
					}
				}

				window.platform.iconified = iconified
				window.inputWindowIconify(iconified)
			}
		} else if event.xproperty().Atom == _glfw.platformWindow.NET_WM_STATE {
			maximized := window.platformWindowMaximized()
			if window.platform.maximized != maximized {
				window.platform.maximized = maximized
				window.inputWindowMaximize(maximized)
			}
		}

		return nil

	case _DestroyNotify:
		return nil
	}

	return nil
}

// getWindowPropertyX11 retrieves a single window property of the specified
// type. The returned data must be released with xFree.
// Inspired by fghGetWindowProperty from freeglut
func getWindowPropertyX11(window _XID, property _Atom, typ _Atom, value *uintptr) _Culong {
	var actualType _Atom
	var actualFormat int32
	var itemCount, bytesAfter _Culong

	xGetWindowProperty(_glfw.platformWindow.display,
		window,
		property,
		0,
		math.MaxInt,
		false,
		typ,
		&actualType,
		&actualFormat,
		&itemCount,
		&bytesAfter,
		value)

	return itemCount
}

// isVisualTransparentX11 reports whether the visual supports framebuffer
// transparency.
func isVisualTransparentX11(visual uintptr) bool {
	if !_glfw.platformWindow.xrender.available {
		return false
	}

	pfPtr := _glfw.platformWindow.xrender.FindVisualFormat(_glfw.platformWindow.display, visual)
	if pfPtr == 0 {
		return false
	}
	return (*_XRenderPictFormat)(unsafe.Pointer(pfPtr)).Direct.AlphaMask != 0
}

// pushSelectionToManagerX11 transfers the owned selections to the clipboard
// manager, so that the contents survive the application exiting.
func pushSelectionToManagerX11() {
	xConvertSelection(_glfw.platformWindow.display,
		_glfw.platformWindow.CLIPBOARD_MANAGER,
		_glfw.platformWindow.SAVE_TARGETS,
		_None,
		_glfw.platformWindow.helperWindowHandle,
		_CurrentTime)

	if isSelectionEventCallback == 0 {
		isSelectionEventCallback = purego.NewCallback(isSelectionEvent)
	}

	for {
		var event _XEvent

		for xCheckIfEvent(_glfw.platformWindow.display, &event, isSelectionEventCallback, 0) {
			switch event.EventType() {
			case _SelectionRequest:
				handleSelectionRequest(&event)

			case _SelectionNotify:
				if event.xselection().Target == _glfw.platformWindow.SAVE_TARGETS {
					// This means one of two things; either the selection
					// was not owned, which means there is no clipboard
					// manager, or the transfer to the clipboard manager has
					// completed
					// In either case, it means the work here is done
					return
				}
			}
		}

		waitForX11Event(nil)
	}
}

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig_ *fbconfig) error {
	var visual uintptr
	var depth int32

	if ctxconfig.client != NoAPI {
		switch ctxconfig.source {
		case NativeContextAPI:
			if err := initGLX(); err != nil {
				return err
			}
			var err error
			visual, depth, err = chooseVisualGLX(wndconfig, ctxconfig, fbconfig_)
			if err != nil {
				return err
			}
		case EGLContextAPI:
			if err := initEGL(); err != nil {
				return err
			}
			var err error
			visual, depth, err = chooseVisualEGL(wndconfig, ctxconfig, fbconfig_)
			if err != nil {
				return err
			}
		case OSMesaContextAPI:
			return fmt.Errorf("glfw: x11: OSMesa is not supported: %w", APIUnavailable)
		}
	}

	if visual == 0 {
		visual = xDefaultVisual(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen))
		depth = xDefaultDepth(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen))
	}

	if err := createNativeWindow(w, wndconfig, visual, depth); err != nil {
		return err
	}

	if ctxconfig.client != NoAPI {
		switch ctxconfig.source {
		case NativeContextAPI:
			if err := createContextGLX(w, ctxconfig, fbconfig_); err != nil {
				return err
			}
		case EGLContextAPI:
			if err := createContextEGL(w, ctxconfig, fbconfig_); err != nil {
				return err
			}
		}

		if err := w.refreshContextAttribs(ctxconfig); err != nil {
			return err
		}
	}

	if wndconfig.mousePassthrough {
		if err := w.platformSetWindowMousePassthrough(true); err != nil {
			return err
		}
	}

	if w.monitor != nil {
		w.platformShowWindow()
		updateWindowMode(w)
		if err := acquireMonitor(w); err != nil {
			return err
		}

		if wndconfig.centerCursor {
			if err := w.centerCursorInContentArea(); err != nil {
				return err
			}
		}
	} else {
		if wndconfig.visible {
			w.platformShowWindow()
			if wndconfig.focused {
				if err := w.platformFocusWindow(); err != nil {
					return err
				}
			}
		}
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

// packXSyncValue packs an X Sync value's high and low halves into the machine
// word that a by-value XSyncValue occupies as a function argument.
func packXSyncValue(hi int32, lo uint32) uint64 {
	if cpu.IsBigEndian {
		return uint64(uint32(hi))<<32 | uint64(lo)
	}
	return uint64(uint32(hi)) | uint64(lo)<<32
}

// signalFrameSyncCounter acknowledges the most recent _NET_WM_SYNC_REQUEST by
// setting the sync counter to the requested value, telling the window manager a
// frame has been drawn at the new size. It must be called right after a buffer
// swap, and does nothing unless a request is pending.
func (w *Window) signalFrameSyncCounter() {
	if !w.platform.syncRequested.Swap(false) {
		return
	}

	packed := w.platform.syncValue.Load()
	_glfw.platformWindow.xsync.SetCounter(_glfw.platformWindow.display,
		w.platform.syncCounter, packXSyncValue(int32(uint32(packed>>32)), uint32(packed)))
	// Flush so the compositor sees the new value promptly: at a low frame rate
	// the swap above can be the last X request for a while.
	xFlush(_glfw.platformWindow.display)
}

func (w *Window) platformDestroyWindow() error {
	if _glfw.platformWindow.disabledCursorWindow == w {
		if err := enableCursor(w); err != nil {
			return err
		}
	}

	if w.monitor != nil {
		releaseMonitor(w)
	}

	if w.platform.ic != 0 {
		xDestroyIC(w.platform.ic)
		w.platform.ic = 0
	}

	if w.platform.syncCounter != 0 {
		_glfw.platformWindow.xsync.DestroyCounter(_glfw.platformWindow.display, w.platform.syncCounter)
		w.platform.syncCounter = 0
	}

	if w.context.destroy != nil {
		if err := w.context.destroy(w); err != nil {
			return err
		}
	}

	if w.platform.handle != 0 {
		delete(_glfw.platformWindow.windowsByXID, w.platform.handle)
		xUnmapWindow(_glfw.platformWindow.display, w.platform.handle)
		xDestroyWindow(_glfw.platformWindow.display, w.platform.handle)
		w.platform.handle = 0
	}

	if w.platform.colormap != 0 {
		xFreeColormap(_glfw.platformWindow.display, w.platform.colormap)
		w.platform.colormap = 0
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowTitle(title string) error {
	xutf8SetWMProperties(_glfw.platformWindow.display,
		w.platform.handle,
		title, title,
		0, 0,
		0, 0, 0)

	titleBytes := []byte(title)
	var titlePtr unsafe.Pointer
	if len(titleBytes) > 0 {
		titlePtr = unsafe.Pointer(&titleBytes[0])
	}

	xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
		_glfw.platformWindow.NET_WM_NAME, _glfw.platformWindow.UTF8_STRING, 8,
		_PropModeReplace,
		titlePtr, int32(len(titleBytes)))

	xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
		_glfw.platformWindow.NET_WM_ICON_NAME, _glfw.platformWindow.UTF8_STRING, 8,
		_PropModeReplace,
		titlePtr, int32(len(titleBytes)))

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	if len(images) > 0 {
		var longCount int
		for _, image := range images {
			longCount += 2 + image.Width*image.Height
		}

		icon := make([]_Culong, 0, longCount)

		for _, image := range images {
			icon = append(icon, _Culong(image.Width), _Culong(image.Height))

			for j := 0; j < image.Width*image.Height; j++ {
				icon = append(icon,
					_Culong(image.Pixels[j*4+0])<<16|
						_Culong(image.Pixels[j*4+1])<<8|
						_Culong(image.Pixels[j*4+2])<<0|
						_Culong(image.Pixels[j*4+3])<<24)
			}
		}

		// NOTE: XChangeProperty expects 32-bit values like the image data above to be
		//       placed in the 32 least significant bits of individual longs.  This is
		//       true even if long is 64-bit and a WM protocol calls for "packed" data.
		//       This is because of a historical mistake that then became part of the Xlib
		//       ABI.  Xlib will pack these values into a regular array of 32-bit values
		//       before sending it over the wire.
		xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
			_glfw.platformWindow.NET_WM_ICON,
			_XA_CARDINAL, 32,
			_PropModeReplace,
			unsafe.Pointer(&icon[0]),
			int32(len(icon)))
	} else {
		xDeleteProperty(_glfw.platformWindow.display, w.platform.handle,
			_glfw.platformWindow.NET_WM_ICON)
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	var dummy _XID
	var x, y int32

	xTranslateCoordinates(_glfw.platformWindow.display, w.platform.handle, _glfw.platformWindow.root,
		0, 0, &x, &y, &dummy)

	return int(x), int(y), nil
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	// HACK: Explicitly setting PPosition to any value causes some WMs, notably
	//       Compiz and Metacity, to honor the position of unmapped windows
	if !w.platformWindowVisible() {
		var supplied _Clong
		hintsPtr := xAllocSizeHints()
		hints := (*_XSizeHints)(unsafe.Pointer(hintsPtr))

		if xGetWMNormalHints(_glfw.platformWindow.display, w.platform.handle, hints, &supplied) != 0 {
			hints.Flags |= _PPosition
			hints.X = 0
			hints.Y = 0

			xSetWMNormalHints(_glfw.platformWindow.display, w.platform.handle, hints)
		}

		xFree(hintsPtr)
	}

	xMoveWindow(_glfw.platformWindow.display, w.platform.handle, int32(xpos), int32(ypos))
	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	var attribs _XWindowAttributes
	xGetWindowAttributes(_glfw.platformWindow.display, w.platform.handle, &attribs)

	return int(attribs.Width), int(attribs.Height), nil
}

func (w *Window) platformSetWindowSize(width, height int) error {
	if w.monitor != nil {
		if w.monitor.window == w {
			if err := acquireMonitor(w); err != nil {
				return err
			}
		}
	} else {
		if !w.resizable {
			updateNormalHints(w, width, height)
		}

		xResizeWindow(_glfw.platformWindow.display, w.platform.handle, uint32(width), uint32(height))
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return err
	}
	updateNormalHints(w, width, height)
	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return err
	}
	updateNormalHints(w, width, height)
	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	return w.platformGetWindowSize()
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	if w.monitor != nil || !w.decorated {
		return 0, 0, 0, 0, nil
	}

	if _glfw.platformWindow.NET_FRAME_EXTENTS == _None {
		return 0, 0, 0, 0, nil
	}

	if !w.platformWindowVisible() && _glfw.platformWindow.NET_REQUEST_FRAME_EXTENTS != 0 {
		var event _XEvent
		timeout := 0.5

		// Ensure _NET_FRAME_EXTENTS is set, allowing glfwGetWindowFrameSize to
		// function before the window is mapped
		sendEventToWM(w, _glfw.platformWindow.NET_REQUEST_FRAME_EXTENTS, 0, 0, 0, 0, 0)

		// HACK: Use a timeout because earlier versions of some window managers
		//       (at least Unity, Fluxbox and Xfwm) failed to send the reply
		//       They have been fixed but broken versions are still in the wild
		//       If you are affected by this and your window manager is NOT
		//       listed above, PLEASE report it to their and our issue trackers
		if isFrameExtentsEventCallback == 0 {
			isFrameExtentsEventCallback = purego.NewCallback(isFrameExtentsEvent)
		}
		for !xCheckIfEvent(_glfw.platformWindow.display,
			&event,
			isFrameExtentsEventCallback,
			uintptr(w.platform.handle)) {
			if !waitForX11Event(&timeout) {
				return 0, 0, 0, 0, fmt.Errorf("glfw: x11: the window manager has a broken _NET_REQUEST_FRAME_EXTENTS implementation; please report this issue: %w", PlatformError)
			}
		}
	}

	var extentsPtr uintptr
	if getWindowPropertyX11(w.platform.handle,
		_glfw.platformWindow.NET_FRAME_EXTENTS,
		_XA_CARDINAL,
		&extentsPtr) == 4 {
		extents := unsafe.Slice((*_Clong)(unsafe.Pointer(extentsPtr)), 4)
		left = int(extents[0])
		top = int(extents[2])
		right = int(extents[1])
		bottom = int(extents[3])
	}

	if extentsPtr != 0 {
		xFree(extentsPtr)
	}

	return left, top, right, bottom, nil
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	return _glfw.platformWindow.contentScaleX, _glfw.platformWindow.contentScaleY, nil
}

func (w *Window) platformIconifyWindow() {
	if w.platform.overrideRedirect {
		// Override-redirect windows cannot be iconified or restored, as those
		// tasks are performed by the window manager
		return
	}

	xIconifyWindow(_glfw.platformWindow.display, w.platform.handle, int32(_glfw.platformWindow.screen))
	xFlush(_glfw.platformWindow.display)
}

func (w *Window) platformRestoreWindow() {
	if w.platform.overrideRedirect {
		// Override-redirect windows cannot be iconified or restored, as those
		// tasks are performed by the window manager
		return
	}

	if w.platformWindowIconified() {
		xMapWindow(_glfw.platformWindow.display, w.platform.handle)
		waitForVisibilityNotify(w)
	} else if w.platformWindowVisible() {
		if _glfw.platformWindow.NET_WM_STATE != 0 &&
			_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT != 0 &&
			_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ != 0 {
			sendEventToWM(w,
				_glfw.platformWindow.NET_WM_STATE,
				_NET_WM_STATE_REMOVE,
				int(_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT),
				int(_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ),
				1, 0)
		}
	}

	xFlush(_glfw.platformWindow.display)
}

func (w *Window) platformMaximizeWindow() error {
	if _glfw.platformWindow.NET_WM_STATE == 0 ||
		_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT == 0 ||
		_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ == 0 {
		return nil
	}

	if w.platformWindowVisible() {
		sendEventToWM(w,
			_glfw.platformWindow.NET_WM_STATE,
			_NET_WM_STATE_ADD,
			int(_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT),
			int(_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ),
			1, 0)
	} else {
		var statesPtr uintptr
		count := getWindowPropertyX11(w.platform.handle,
			_glfw.platformWindow.NET_WM_STATE,
			_XA_ATOM,
			&statesPtr)

		// NOTE: We don't check for failure as this property may not exist yet
		//       and that's fine (and we'll create it implicitly with append)

		missing := []_Atom{
			_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT,
			_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ,
		}

		if statesPtr != 0 {
			states := unsafe.Slice((*_Atom)(unsafe.Pointer(statesPtr)), int(count))
			for _, state := range states {
				for j := 0; j < len(missing); j++ {
					if state == missing[j] {
						missing[j] = missing[len(missing)-1]
						missing = missing[:len(missing)-1]
					}
				}
			}

			xFree(statesPtr)
		}

		if len(missing) == 0 {
			return nil
		}

		xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
			_glfw.platformWindow.NET_WM_STATE, _XA_ATOM, 32,
			_PropModeAppend,
			unsafe.Pointer(&missing[0]),
			int32(len(missing)))
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformShowWindow() {
	if w.platformWindowVisible() {
		return
	}

	xMapWindow(_glfw.platformWindow.display, w.platform.handle)
	waitForVisibilityNotify(w)
}

func (w *Window) platformHideWindow() {
	xUnmapWindow(_glfw.platformWindow.display, w.platform.handle)
	xFlush(_glfw.platformWindow.display)
}

func (w *Window) platformRequestWindowAttention() {
	if _glfw.platformWindow.NET_WM_STATE == 0 || _glfw.platformWindow.NET_WM_STATE_DEMANDS_ATTENTION == 0 {
		return
	}

	sendEventToWM(w,
		_glfw.platformWindow.NET_WM_STATE,
		_NET_WM_STATE_ADD,
		int(_glfw.platformWindow.NET_WM_STATE_DEMANDS_ATTENTION),
		0, 1, 0)
}

func (w *Window) platformFocusWindow() error {
	if _glfw.platformWindow.NET_ACTIVE_WINDOW != 0 {
		sendEventToWM(w, _glfw.platformWindow.NET_ACTIVE_WINDOW, 1, 0, 0, 0, 0)
	} else if w.platformWindowVisible() {
		xRaiseWindow(_glfw.platformWindow.display, w.platform.handle)
		xSetInputFocus(_glfw.platformWindow.display, w.platform.handle, _RevertToParent, _CurrentTime)
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	if w.monitor == monitor {
		if monitor != nil {
			if monitor.window == w {
				if err := acquireMonitor(w); err != nil {
					return err
				}
			}
		} else {
			if !w.resizable {
				updateNormalHints(w, width, height)
			}

			xMoveResizeWindow(_glfw.platformWindow.display, w.platform.handle,
				int32(xpos), int32(ypos), uint32(width), uint32(height))
		}

		xFlush(_glfw.platformWindow.display)
		return nil
	}

	if w.monitor != nil {
		if err := w.platformSetWindowDecorated(w.decorated); err != nil {
			return err
		}
		if err := w.platformSetWindowFloating(w.floating); err != nil {
			return err
		}
		releaseMonitor(w)
	}

	w.inputWindowMonitor(monitor)
	updateNormalHints(w, width, height)

	if w.monitor != nil {
		if !w.platformWindowVisible() {
			xMapRaised(_glfw.platformWindow.display, w.platform.handle)
			waitForVisibilityNotify(w)
		}

		updateWindowMode(w)
		if err := acquireMonitor(w); err != nil {
			return err
		}
	} else {
		updateWindowMode(w)
		xMoveResizeWindow(_glfw.platformWindow.display, w.platform.handle,
			int32(xpos), int32(ypos), uint32(width), uint32(height))
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformWindowFocused() bool {
	var focused _XID
	var state int32

	xGetInputFocus(_glfw.platformWindow.display, &focused, &state)
	return w.platform.handle == focused
}

func (w *Window) platformWindowIconified() bool {
	return getWindowState(w) == _IconicState
}

func (w *Window) platformWindowVisible() bool {
	var wa _XWindowAttributes
	xGetWindowAttributes(_glfw.platformWindow.display, w.platform.handle, &wa)
	return wa.MapState == _IsViewable
}

func (w *Window) platformWindowMaximized() bool {
	if _glfw.platformWindow.NET_WM_STATE == 0 ||
		_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT == 0 ||
		_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ == 0 {
		return false
	}

	var statesPtr uintptr
	count := getWindowPropertyX11(w.platform.handle,
		_glfw.platformWindow.NET_WM_STATE,
		_XA_ATOM,
		&statesPtr)

	maximized := false
	if statesPtr != 0 {
		states := unsafe.Slice((*_Atom)(unsafe.Pointer(statesPtr)), int(count))
		for _, state := range states {
			if state == _glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT ||
				state == _glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ {
				maximized = true
				break
			}
		}

		xFree(statesPtr)
	}

	return maximized
}

func (w *Window) platformWindowHovered() (bool, error) {
	window := _glfw.platformWindow.root
	for window != 0 {
		var root, child _XID
		var rootX, rootY, childX, childY int32
		var mask uint32

		grabErrorHandlerX11()

		result := xQueryPointer(_glfw.platformWindow.display, window,
			&root, &child, &rootX, &rootY,
			&childX, &childY, &mask)

		releaseErrorHandlerX11()

		if _glfw.platformWindow.errorCode == _BadWindow {
			window = _glfw.platformWindow.root
		} else if !result {
			return false, nil
		} else if child == w.platform.handle {
			return true, nil
		} else {
			window = child
		}
	}

	return false, nil
}

func (w *Window) platformFramebufferTransparent() bool {
	if !w.platform.transparent {
		return false
	}

	return xGetSelectionOwner(_glfw.platformWindow.display, _glfw.platformWindow.NET_WM_CM_Sx) != _None
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return err
	}
	updateNormalHints(w, width, height)
	return nil
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	updateWindowHints(w)
	return nil
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	if _glfw.platformWindow.NET_WM_STATE == 0 || _glfw.platformWindow.NET_WM_STATE_ABOVE == 0 {
		return nil
	}

	if w.platformWindowVisible() {
		action := _NET_WM_STATE_REMOVE
		if enabled {
			action = _NET_WM_STATE_ADD
		}
		sendEventToWM(w,
			_glfw.platformWindow.NET_WM_STATE,
			action,
			int(_glfw.platformWindow.NET_WM_STATE_ABOVE),
			0, 1, 0)
	} else {
		var statesPtr uintptr
		count := getWindowPropertyX11(w.platform.handle,
			_glfw.platformWindow.NET_WM_STATE,
			_XA_ATOM,
			&statesPtr)

		// NOTE: We don't check for failure as this property may not exist yet
		//       and that's fine (and we'll create it implicitly with append)

		var states []_Atom
		if statesPtr != 0 {
			states = unsafe.Slice((*_Atom)(unsafe.Pointer(statesPtr)), int(count))
		}

		if enabled {
			i := 0
			for ; i < len(states); i++ {
				if states[i] == _glfw.platformWindow.NET_WM_STATE_ABOVE {
					break
				}
			}

			if i == len(states) {
				above := _glfw.platformWindow.NET_WM_STATE_ABOVE
				xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
					_glfw.platformWindow.NET_WM_STATE, _XA_ATOM, 32,
					_PropModeAppend,
					unsafe.Pointer(&above),
					1)
			}
		} else if states != nil {
			i := 0
			for ; i < len(states); i++ {
				if states[i] == _glfw.platformWindow.NET_WM_STATE_ABOVE {
					break
				}
			}

			if i < len(states) {
				states[i] = states[len(states)-1]
				states = states[:len(states)-1]

				var statesHead unsafe.Pointer
				if len(states) > 0 {
					statesHead = unsafe.Pointer(&states[0])
				}
				xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
					_glfw.platformWindow.NET_WM_STATE, _XA_ATOM, 32,
					_PropModeReplace, statesHead, int32(len(states)))
			}
		}

		if statesPtr != 0 {
			xFree(statesPtr)
		}
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetWindowMousePassthrough(enabled bool) error {
	if !_glfw.platformWindow.xshape.available {
		return nil
	}

	if enabled {
		region := xCreateRegion()
		_glfw.platformWindow.xshape.CombineRegion(_glfw.platformWindow.display, w.platform.handle,
			_ShapeInput, 0, 0, region, _ShapeSet)
		xDestroyRegion(region)
	} else {
		_glfw.platformWindow.xshape.CombineMask(_glfw.platformWindow.display, w.platform.handle,
			_ShapeInput, 0, 0, _None, _ShapeSet)
	}
	return nil
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	opacity := float32(1)

	if xGetSelectionOwner(_glfw.platformWindow.display, _glfw.platformWindow.NET_WM_CM_Sx) != 0 {
		var valuePtr uintptr
		if getWindowPropertyX11(w.platform.handle,
			_glfw.platformWindow.NET_WM_WINDOW_OPACITY,
			_XA_CARDINAL,
			&valuePtr) != 0 {
			value := uint32(*(*_Culong)(unsafe.Pointer(valuePtr)))
			opacity = float32(float64(value) / 0xffffffff)
		}

		if valuePtr != 0 {
			xFree(valuePtr)
		}
	}

	return opacity, nil
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	value := _Culong(0xffffffff * float64(opacity))
	xChangeProperty(_glfw.platformWindow.display, w.platform.handle,
		_glfw.platformWindow.NET_WM_WINDOW_OPACITY, _XA_CARDINAL, 32,
		_PropModeReplace, unsafe.Pointer(&value), 1)
	return nil
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	if !_glfw.platformWindow.xi.available {
		return nil
	}

	if _glfw.platformWindow.disabledCursorWindow != w {
		return nil
	}

	if enabled {
		enableRawMouseMotion(w)
	} else {
		disableRawMouseMotion(w)
	}
	return nil
}

func platformRawMouseMotionSupported() bool {
	return _glfw.platformWindow.xi.available
}

func platformPollEvents() error {
	drainEmptyEvents()

	xPending(_glfw.platformWindow.display)

	for xQLength(_glfw.platformWindow.display) != 0 {
		var event _XEvent
		xNextEvent(_glfw.platformWindow.display, &event)
		if err := processEvent(&event); err != nil {
			return err
		}
	}

	if window := _glfw.platformWindow.disabledCursorWindow; window != nil {
		width, height, err := window.platformGetWindowSize()
		if err != nil {
			return err
		}

		// NOTE: Re-center the cursor only if it has moved since the last call,
		//       to avoid breaking glfwWaitEvents with MotionNotify
		if window.platform.lastCursorPosX != width/2 ||
			window.platform.lastCursorPosY != height/2 {
			if err := window.platformSetCursorPos(float64(width/2), float64(height/2)); err != nil {
				return err
			}
		}
	}

	xFlush(_glfw.platformWindow.display)
	return nil
}

func platformWaitEvents() error {
	waitForAnyEvent(nil)
	return platformPollEvents()
}

func platformWaitEventsTimeout(timeout float64) error {
	waitForAnyEvent(&timeout)
	return platformPollEvents()
}

func platformPostEmptyEvent() error {
	writeEmptyEvent()
	return nil
}

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	var root, child _XID
	var rootX, rootY, childX, childY int32
	var mask uint32

	xQueryPointer(_glfw.platformWindow.display, w.platform.handle,
		&root, &child,
		&rootX, &rootY, &childX, &childY,
		&mask)

	return float64(childX), float64(childY), nil
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	// Store the new position so it can be recognized later
	w.platform.warpCursorPosX = int(xpos)
	w.platform.warpCursorPosY = int(ypos)

	xWarpPointer(_glfw.platformWindow.display, _None, w.platform.handle,
		0, 0, 0, 0, int32(xpos), int32(ypos))
	xFlush(_glfw.platformWindow.display)
	return nil
}

func (w *Window) platformSetCursorMode(mode int) error {
	if w.platformWindowFocused() {
		if mode == CursorDisabled {
			xpos, ypos, err := w.platformGetCursorPos()
			if err != nil {
				return err
			}
			_glfw.platformWindow.restoreCursorPosX = xpos
			_glfw.platformWindow.restoreCursorPosY = ypos
			if err := w.centerCursorInContentArea(); err != nil {
				return err
			}
			if w.rawMouseMotion {
				enableRawMouseMotion(w)
			}
		} else if _glfw.platformWindow.disabledCursorWindow == w {
			if w.rawMouseMotion {
				disableRawMouseMotion(w)
			}
		}

		if mode == CursorDisabled {
			captureCursor(w)
		} else {
			releaseCursor()
		}

		if mode == CursorDisabled {
			_glfw.platformWindow.disabledCursorWindow = w
		} else if _glfw.platformWindow.disabledCursorWindow == w {
			_glfw.platformWindow.disabledCursorWindow = nil
			if err := w.platformSetCursorPos(_glfw.platformWindow.restoreCursorPosX,
				_glfw.platformWindow.restoreCursorPosY); err != nil {
				return err
			}
		}
	}

	updateCursorImage(w)
	xFlush(_glfw.platformWindow.display)
	return nil
}

func platformGetScancodeName(scancode int) (string, error) {
	if !_glfw.platformWindow.xkb.available {
		return "", nil
	}

	if scancode < 0 || scancode > 0xff {
		return "", fmt.Errorf("glfw: x11: invalid scancode %d: %w", scancode, InvalidValue)
	}

	key := _glfw.platformWindow.keycodes[scancode]
	if key == KeyUnknown {
		return "", nil
	}

	keysym := xkbKeycodeToKeysym(_glfw.platformWindow.display,
		uint32(scancode), int32(_glfw.platformWindow.xkb.group), 0)
	if keysym == _NoSymbol {
		return "", nil
	}

	codepoint, ok := keySym2Unicode(uint32(keysym))
	if !ok {
		return "", nil
	}

	_glfw.platformWindow.keynames[key] = string(codepoint)
	return _glfw.platformWindow.keynames[key], nil
}

func platformGetKeyScancode(key Key) int {
	return _glfw.platformWindow.scancodes[key]
}

func (c *Cursor) platformCreateCursor(img *image.NRGBA, xhot, yhot int) error {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	// Repack into a tight RGBA buffer: createCursorX11 expects contiguous
	// pixels, and a non-zero image origin or non-trivial stride would
	// confuse it.
	pixels := make([]byte, w*h*4)
	for y := range h {
		src := img.PixOffset(b.Min.X, b.Min.Y+y)
		copy(pixels[y*w*4:(y+1)*w*4], img.Pix[src:src+w*4])
	}

	c.platform.handle = createCursorX11(&Image{Width: w, Height: h, Pixels: pixels}, xhot, yhot)
	if c.platform.handle == _None {
		return fmt.Errorf("glfw: x11: failed to create cursor: %w", PlatformError)
	}

	return nil
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	// See GLFW 3.4 implementation.
	if _glfw.platformWindow.xcursor.handle != 0 {
		theme := _glfw.platformWindow.xcursor.GetTheme(_glfw.platformWindow.display)
		if theme != 0 {
			size := _glfw.platformWindow.xcursor.GetDefaultSize(_glfw.platformWindow.display)
			var name string

			switch shape {
			case ArrowCursor:
				name = "default"
			case IBeamCursor:
				name = "text"
			case CrosshairCursor:
				name = "crosshair"
			case HandCursor:
				name = "pointer"
			case HResizeCursor:
				name = "ew-resize"
			case VResizeCursor:
				name = "ns-resize"
			case ResizeNWSECursor:
				name = "nwse-resize"
			case ResizeNESWCursor:
				name = "nesw-resize"
			case ResizeAllCursor:
				name = "all-scroll"
			case NotAllowedCursor:
				name = "not-allowed"
			}

			imagePtr := _glfw.platformWindow.xcursor.LibraryLoadImage(name, theme, size)
			if imagePtr != 0 {
				c.platform.handle = _glfw.platformWindow.xcursor.ImageLoadCursor(
					_glfw.platformWindow.display, imagePtr)
				_glfw.platformWindow.xcursor.ImageDestroy(imagePtr)
			}
		}
	}

	if c.platform.handle == _None {
		var native uint32

		switch shape {
		case ArrowCursor:
			native = _XC_left_ptr
		case IBeamCursor:
			native = _XC_xterm
		case CrosshairCursor:
			native = _XC_crosshair
		case HandCursor:
			native = _XC_hand2
		case HResizeCursor:
			native = _XC_sb_h_double_arrow
		case VResizeCursor:
			native = _XC_sb_v_double_arrow
		case ResizeAllCursor:
			native = _XC_fleur
		default:
			// An unavailable shape is deliberately not an error: callers
			// fall back to the default cursor (#2476).
			return nil
		}

		c.platform.handle = xCreateFontCursor(_glfw.platformWindow.display, native)
		if c.platform.handle == _None {
			return fmt.Errorf("glfw: x11: failed to create standard cursor: %w", PlatformError)
		}
	}

	return nil
}

func (c *Cursor) platformDestroyCursor() error {
	if c.platform.handle != _None {
		xFreeCursor(_glfw.platformWindow.display, c.platform.handle)
	}
	return nil
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	if w.cursorMode == CursorNormal {
		updateCursorImage(w)
		xFlush(_glfw.platformWindow.display)
	}
	return nil
}

func platformSetClipboardString(str string) error {
	_glfw.platformWindow.clipboardString = str

	xSetSelectionOwner(_glfw.platformWindow.display,
		_glfw.platformWindow.CLIPBOARD,
		_glfw.platformWindow.helperWindowHandle,
		_CurrentTime)

	if xGetSelectionOwner(_glfw.platformWindow.display, _glfw.platformWindow.CLIPBOARD) !=
		_glfw.platformWindow.helperWindowHandle {
		return fmt.Errorf("glfw: x11: failed to become owner of clipboard selection: %w", PlatformError)
	}
	return nil
}

func platformGetClipboardString() (string, error) {
	return getSelectionString(_glfw.platformWindow.CLIPBOARD)
}

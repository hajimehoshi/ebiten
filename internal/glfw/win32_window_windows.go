// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

func (w *Window) getWindowStyle() uint32 {
	var style uint32 = _WS_CLIPSIBLINGS | _WS_CLIPCHILDREN

	if w.monitor != nil {
		style |= _WS_POPUP
	} else {
		style |= _WS_SYSMENU | _WS_MINIMIZEBOX
		if w.decorated {
			style |= _WS_CAPTION
			if w.resizable {
				style |= _WS_THICKFRAME
				if w.maxwidth == DontCare && w.maxheight == DontCare {
					style |= _WS_MAXIMIZEBOX
				}
			}
		} else {
			style |= _WS_POPUP
		}
	}

	return style
}

func (w *Window) getWindowExStyle() uint32 {
	var style uint32 = _WS_EX_APPWINDOW

	if w.floating {
		style |= _WS_EX_TOPMOST
	}

	return style
}

func chooseImage(images []*Image, width, height int) *Image {
	var leastDiff uint = math.MaxUint32
	var closest *Image
	for _, image := range images {
		currDiff := abs(image.Width*image.Height - width*height)
		if currDiff < leastDiff {
			closest = image
			leastDiff = currDiff
		}
	}
	return closest
}

func createIcon(image *Image, xhot, yhot int, icon bool) (_HICON, error) {
	var bi _BITMAPV5HEADER
	bi.bV5Size = uint32(unsafe.Sizeof(bi))
	bi.bV5Width = int32(image.Width)
	bi.bV5Height = int32(-image.Height)
	bi.bV5Planes = 1
	bi.bV5BitCount = 32
	bi.bV5Compression = _BI_BITFIELDS
	bi.bV5RedMask = 0x00ff0000
	bi.bV5GreenMask = 0x0000ff00
	bi.bV5BlueMask = 0x000000ff
	bi.bV5AlphaMask = 0xff000000

	dc, err := _GetDC(0)
	if err != nil {
		return 0, err
	}
	defer _ReleaseDC(0, dc)

	color, targetPtr, err := _CreateDIBSection(dc, &bi, _DIB_RGB_COLORS, 0, 0)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = _DeleteObject(_HGDIOBJ(color))
	}()

	mask, err := _CreateBitmap(int32(image.Width), int32(image.Height), 1, 1, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = _DeleteObject(_HGDIOBJ(mask))
	}()

	source := image.Pixels
	target := unsafe.Slice((*byte)(unsafe.Pointer(targetPtr)), len(source))
	for i := 0; i < len(source)/4; i++ {
		target[4*i] = source[4*i+2]
		target[4*i+1] = source[4*i+1]
		target[4*i+2] = source[4*i+0]
		target[4*i+3] = source[4*i+3]
	}

	var iconInt32 int32
	if icon {
		iconInt32 = 1
	}
	ii := _ICONINFO{
		fIcon:    iconInt32,
		xHotspot: uint32(xhot),
		yHotspot: uint32(yhot),
		hbmMask:  mask,
		hbmColor: color,
	}
	handle, err := _CreateIconIndirect(&ii)
	if err != nil {
		return 0, err
	}

	return handle, nil
}

func (w *Window) applyAspectRatio(edge int, area *_RECT) error {
	var frame _RECT

	ratio := float32(w.numer) / float32(w.denom)
	style := w.getWindowStyle()
	exStyle := w.getWindowExStyle()

	if winver.IsWindows10AnniversaryUpdateOrGreater() {
		if err := _AdjustWindowRectExForDpi(&frame, style, false, exStyle, _GetDpiForWindow(w.platform.handle)); err != nil {
			return err
		}
	} else {
		if err := _AdjustWindowRectEx(&frame, style, false, exStyle); err != nil {
			return err
		}
	}

	if edge == _WMSZ_LEFT || edge == _WMSZ_BOTTOMLEFT || edge == _WMSZ_RIGHT || edge == _WMSZ_BOTTOMRIGHT {
		area.bottom = area.top + int32(frame.bottom-frame.top) + int32(float32(area.right-area.left-int32(frame.right-frame.left))/ratio)
	} else if edge == _WMSZ_TOPLEFT || edge == _WMSZ_TOPRIGHT {
		area.top = area.bottom - int32(frame.bottom-frame.top) - int32(float32(area.right-area.left-int32(frame.right-frame.left))/ratio)
	} else if edge == _WMSZ_TOP || edge == _WMSZ_BOTTOM {
		area.right = area.left + int32(frame.right-frame.left) + int32(float32(area.bottom-area.top-int32(frame.bottom-frame.top))*ratio)
	}

	return nil
}

func (w *Window) updateCursorImage() error {
	if w.cursorMode == CursorNormal {
		if w.cursor != nil {
			_SetCursor(w.cursor.platform.handle)
		} else {
			cursor, err := _LoadCursorW(0, _IDC_ARROW)
			if err != nil {
				return err
			}
			_SetCursor(cursor)
		}
	} else {
		// Connected via Remote Desktop, nil cursor will present SetCursorPos the move the cursor.
		// using a blank cursor fix that.
		// When not via Remote Desktop, platformWindow.blankCursor should be nil.
		_SetCursor(_glfw.platformWindow.blankCursor)
	}
	return nil
}

func (w *Window) clientToScreen(rect _RECT) (_RECT, error) {
	point := _POINT{
		x: rect.left,
		y: rect.top,
	}
	if err := _ClientToScreen(w.platform.handle, &point); err != nil {
		return _RECT{}, err
	}
	rect.left = point.x
	rect.top = point.y

	point = _POINT{
		x: rect.right,
		y: rect.bottom,
	}
	if err := _ClientToScreen(w.platform.handle, &point); err != nil {
		return _RECT{}, err
	}
	rect.right = point.x
	rect.bottom = point.y
	return rect, nil
}

func captureCursor(window *Window) error {
	clipRect, err := _GetClientRect(window.platform.handle)
	if err != nil {
		return err
	}
	clipRect, err = window.clientToScreen(clipRect)
	if err != nil {
		return err
	}
	if err := _ClipCursor(&clipRect); err != nil {
		return err
	}
	_glfw.platformWindow.capturedCursorWindow = window
	return nil
}

func releaseCursor() error {
	if err := _ClipCursor(nil); err != nil {
		return err
	}
	_glfw.platformWindow.capturedCursorWindow = nil
	return nil
}

func (w *Window) enableRawMouseMotion() error {
	rid := []_RAWINPUTDEVICE{
		{
			usUsagePage: 0x01,
			usUsage:     0x02,
			dwFlags:     0,
			hwndTarget:  w.platform.handle,
		},
	}
	return _RegisterRawInputDevices(rid)
}

func (w *Window) disableRawMouseMotion() error {
	rid := []_RAWINPUTDEVICE{
		{
			usUsagePage: 0x01,
			usUsage:     0x02,
			dwFlags:     _RIDEV_REMOVE,
			hwndTarget:  0,
		},
	}
	return _RegisterRawInputDevices(rid)
}

func (w *Window) disableCursor() error {
	_glfw.platformWindow.disabledCursorWindow = w
	x, y, err := w.platformGetCursorPos()
	if err != nil {
		return err
	}
	_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY = x, y
	if err := w.updateCursorImage(); err != nil {
		return err
	}
	if err := w.centerCursorInContentArea(); err != nil {
		return err
	}
	if err := captureCursor(w); err != nil {
		return err
	}
	if w.rawMouseMotion {
		if err := w.enableRawMouseMotion(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Window) enableCursor() error {
	if w.rawMouseMotion {
		if err := w.disableRawMouseMotion(); err != nil {
			return err
		}
	}
	_glfw.platformWindow.disabledCursorWindow = nil
	if err := releaseCursor(); err != nil {
		return err
	}
	if err := w.platformSetCursorPos(_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY); err != nil {
		return err
	}
	if err := w.updateCursorImage(); err != nil {
		return err
	}
	return nil
}

func (w *Window) cursorInContentArea() (bool, error) {
	if microsoftgdk.IsXbox() {
		return true, nil
	}

	pos, err := _GetCursorPos()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return false, nil
		}
		return false, err
	}
	if _WindowFromPoint(pos) != w.platform.handle {
		return false, nil
	}
	area, err := _GetClientRect(w.platform.handle)
	if err != nil {
		return false, err
	}
	area, err = w.clientToScreen(area)
	if err != nil {
		return false, err
	}
	return _PtInRect(&area, pos), nil
}

func (w *Window) updateWindowStyles() error {
	s, err := _GetWindowLongW(w.platform.handle, _GWL_STYLE)
	if err != nil {
		return err
	}
	style := uint32(s)
	style &^= _WS_OVERLAPPEDWINDOW | _WS_POPUP
	style |= w.getWindowStyle()

	rect, err := _GetClientRect(w.platform.handle)
	if err != nil {
		return err
	}

	if winver.IsWindows10AnniversaryUpdateOrGreater() {
		if err := _AdjustWindowRectExForDpi(&rect, style, false, w.getWindowExStyle(), _GetDpiForWindow(w.platform.handle)); err != nil {
			return err
		}
	} else {
		if err := _AdjustWindowRectEx(&rect, style, false, w.getWindowExStyle()); err != nil {
			return err
		}
	}

	rect, err = w.clientToScreen(rect)
	if err != nil {
		return err
	}
	if _, err := _SetWindowLongW(w.platform.handle, _GWL_STYLE, int32(style)); err != nil {
		return err
	}
	if err := _SetWindowPos(w.platform.handle, _HWND_TOP, rect.left, rect.top, rect.right-rect.left, rect.bottom-rect.top, _SWP_FRAMECHANGED|_SWP_NOACTIVATE|_SWP_NOZORDER); err != nil {
		return err
	}

	return nil
}

func (w *Window) updateFramebufferTransparency() error {
	if !winver.IsWindowsVistaOrGreater() {
		return nil
	}

	composition, err := _DwmIsCompositionEnabled()
	if err != nil {
		// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
		return nil
	}
	if !composition {
		return nil
	}

	var opaque bool
	if !winver.IsWindows8OrGreater() {
		_, opaque, err = _DwmGetColorizationColor()
		if err != nil {
			// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
			return nil
		}
	}

	if winver.IsWindows8OrGreater() || !opaque {
		region, err := _CreateRectRgn(0, 0, -1, -1)
		if err != nil {
			return err
		}
		defer func() {
			_ = _DeleteObject(_HGDIOBJ(region))
		}()

		bb := _DWM_BLURBEHIND{
			dwFlags:  _DWM_BB_ENABLE | _DWM_BB_BLURREGION,
			hRgnBlur: region,
			fEnable:  1, // true
		}

		// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
		_ = _DwmEnableBlurBehindWindow(w.platform.handle, &bb)
	} else {
		// HACK: Disable framebuffer transparency on Windows 7 when the
		//       colorization color is opaque, because otherwise the window
		//       contents is blended additively with the previous frame instead
		//       of replacing it
		bb := _DWM_BLURBEHIND{
			dwFlags: _DWM_BB_ENABLE,
		}

		// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
		_ = _DwmEnableBlurBehindWindow(w.platform.handle, &bb)
	}
	return nil
}

func getKeyMods() ModifierKey {
	var mods ModifierKey
	if uint16(_GetKeyState(_VK_SHIFT))&0x8000 != 0 {
		mods |= ModShift
	}
	if uint16(_GetKeyState(_VK_CONTROL))&0x8000 != 0 {
		mods |= ModControl
	}
	if uint16(_GetKeyState(_VK_MENU))&0x8000 != 0 {
		mods |= ModAlt
	}
	if uint16(_GetKeyState(_VK_LWIN)|_GetKeyState(_VK_RWIN))&0x8000 != 0 {
		mods |= ModSuper
	}
	if _GetKeyState(_VK_CAPITAL)&1 != 0 {
		mods |= ModCapsLock
	}
	if _GetKeyState(_VK_NUMLOCK)&1 != 0 {
		mods |= ModNumLock
	}
	return mods
}

func (w *Window) fitToMonitor() error {
	mi, ok := _GetMonitorInfoW(w.monitor.platform.handle)
	if !ok {
		return nil
	}
	var hWndInsertAfter windows.HWND
	if w.floating {
		hWndInsertAfter = _HWND_TOPMOST
	} else {
		hWndInsertAfter = _HWND_NOTOPMOST
	}
	if err := _SetWindowPos(w.platform.handle, hWndInsertAfter,
		mi.rcMonitor.left,
		mi.rcMonitor.top,
		mi.rcMonitor.right-mi.rcMonitor.left,
		mi.rcMonitor.bottom-mi.rcMonitor.top,
		_SWP_NOZORDER|_SWP_NOACTIVATE|_SWP_NOCOPYBITS); err != nil {
		return err
	}
	return nil
}

func (w *Window) acquireMonitor() error {
	if _glfw.platformWindow.acquiredMonitorCount == 0 {
		_SetThreadExecutionState(_ES_CONTINUOUS | _ES_DISPLAY_REQUIRED)

		// HACK: When mouse trails are enabled the cursor becomes invisible when
		//       the OpenGL ICD switches to page flipping
		if winver.IsWindowsXPOrGreater() {
			if err := _SystemParametersInfoW(_SPI_GETMOUSETRAILS, 0, uintptr(unsafe.Pointer(&_glfw.platformWindow.mouseTrailSize)), 0); err != nil {
				return err
			}
			if err := _SystemParametersInfoW(_SPI_SETMOUSETRAILS, 0, 0, 0); err != nil {
				return err
			}
		}
	}

	if w.monitor.window == nil {
		_glfw.platformWindow.acquiredMonitorCount++
	}

	if err := w.monitor.setVideoModeWin32(&w.videoMode); err != nil {
		return err
	}
	w.monitor.inputMonitorWindow(w)
	return nil
}

func (w *Window) releaseMonitor() error {
	if w.monitor.window != w {
		return nil
	}

	_glfw.platformWindow.acquiredMonitorCount--
	if _glfw.platformWindow.acquiredMonitorCount == 0 {
		_SetThreadExecutionState(_ES_CONTINUOUS)

		// HACK: Restore mouse trail length saved in acquireMonitor
		if winver.IsWindowsXPOrGreater() {
			if err := _SystemParametersInfoW(_SPI_SETMOUSETRAILS, _glfw.platformWindow.mouseTrailSize, 0, 0); err != nil {
				return err
			}
		}
	}

	w.monitor.inputMonitorWindow(nil)
	w.monitor.restoreVideoModeWin32()
	return nil
}

func (w *Window) maximizeWindowManually() error {
	mi, _ := _GetMonitorInfoW(_MonitorFromWindow(w.platform.handle, _MONITOR_DEFAULTTONEAREST))

	rect := mi.rcWork

	if w.maxwidth != DontCare && w.maxheight != DontCare {
		if rect.right-rect.left > int32(w.maxwidth) {
			rect.right = rect.left + int32(w.maxwidth)
		}
		if rect.bottom-rect.top > int32(w.maxheight) {
			rect.bottom = rect.top + int32(w.maxheight)
		}
	}

	s, err := _GetWindowLongW(w.platform.handle, _GWL_STYLE)
	if err != nil {
		return err
	}
	style := uint32(s)
	style |= _WS_MAXIMIZE
	if _, err := _SetWindowLongW(w.platform.handle, _GWL_STYLE, int32(style)); err != nil {
		return err
	}

	if w.decorated {
		s, err := _GetWindowLongW(w.platform.handle, _GWL_EXSTYLE)
		if err != nil {
			return err
		}
		exStyle := uint32(s)
		if winver.IsWindows10AnniversaryUpdateOrGreater() {
			dpi := _GetDpiForWindow(w.platform.handle)
			if err := _AdjustWindowRectExForDpi(&rect, style, false, exStyle, dpi); err != nil {
				return err
			}
			m, err := _GetSystemMetricsForDpi(_SM_CYCAPTION, dpi)
			if err != nil {
				return err
			}
			_OffsetRect(&rect, 0, m)
		} else {
			if err := _AdjustWindowRectEx(&rect, style, false, exStyle); err != nil {
				return err
			}
			m, err := _GetSystemMetrics(_SM_CYCAPTION)
			if err != nil {
				return err
			}
			_OffsetRect(&rect, 0, m)
		}

		if rect.bottom > mi.rcWork.bottom {
			rect.bottom = mi.rcWork.bottom
		}
	}

	if err := _SetWindowPos(w.platform.handle, _HWND_TOP,
		rect.left, rect.top, rect.right-rect.left, rect.bottom-rect.top,
		_SWP_NOACTIVATE|_SWP_NOZORDER|_SWP_FRAMECHANGED); err != nil {
		return err
	}

	return nil
}

func windowProc(hWnd windows.HWND, uMsg uint32, wParam _WPARAM, lParam _LPARAM) uintptr /*_LRESULT*/ {
	window := handleToWindow[hWnd]
	if window == nil {
		// This is the message handling for the hidden helper window
		// and for a regular window during its initial creation
		switch uMsg {
		case _WM_NCCREATE:
			if winver.IsWindows10AnniversaryUpdateOrGreater() {
				cs := (*_CREATESTRUCTW)(unsafe.Pointer(lParam))
				wndconfig := (*wndconfig)(cs.lpCreateParams)

				// On per-monitor DPI aware V1 systems, only enable
				// non-client scaling for windows that scale the client area
				// We need WM_GETDPISCALEDSIZE from V2 to keep the client
				// area static when the non-client area is scaled
				if wndconfig != nil && wndconfig.scaleToMonitor {
					if err := _EnableNonClientDpiScaling(hWnd); err != nil {
						_glfw.errors = append(_glfw.errors, err)
						return 0
					}
				}
			}

		case _WM_DISPLAYCHANGE:
			if err := pollMonitorsWin32(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		return uintptr(_DefWindowProcW(hWnd, uMsg, wParam, lParam))
	}

	switch uMsg {
	case _WM_MOUSEACTIVATE:
		// HACK: Postpone cursor disabling when the window was activated by
		//       clicking a caption button
		if _HIWORD(uint32(lParam)) == _WM_LBUTTONDOWN {
			if _LOWORD(uint32(lParam)) != _HTCLIENT {
				window.platform.frameAction = true
			}
		}

	case _WM_CAPTURECHANGED:
		// HACK: Disable the cursor once the caption button action has been
		//       completed or cancelled
		if lParam == 0 && window.platform.frameAction {
			if window.cursorMode == CursorDisabled {
				if err := window.disableCursor(); err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}
			}
			window.platform.frameAction = false
		}

	case _WM_SETFOCUS:
		window.inputWindowFocus(true)

		// HACK: Do not disable cursor while the user is interacting with
		//       a caption button
		if window.platform.frameAction {
			break
		}

		if window.cursorMode == CursorDisabled {
			if err := window.disableCursor(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		return 0

	case _WM_KILLFOCUS:
		if window.cursorMode == CursorDisabled {
			if err := window.enableCursor(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		if window.monitor != nil && window.autoIconify {
			window.platformIconifyWindow()
		}

		window.inputWindowFocus(false)
		return 0

	case _WM_SYSCOMMAND:
		switch wParam & 0xfff0 {
		case _SC_SCREENSAVE, _SC_MONITORPOWER:
			if window.monitor != nil {
				// We are running in full screen mode, so disallow
				// screen saver and screen blanking
				return 0
			} else {
				break
			}
		// User trying to access application menu using ALT?
		case _SC_KEYMENU:
			return 0
		}

	case _WM_CLOSE:
		window.inputWindowCloseRequest()
		return 0

	case _WM_INPUTLANGCHANGE:
		updateKeyNamesWin32()
		return 0

	case _WM_CHAR, _WM_SYSCHAR:
		if wParam >= 0xd800 && wParam <= 0xdbff {
			window.platform.highSurrogate = uint16(wParam)
		} else {
			var codepoint rune

			if wParam >= 0xdc00 && wParam <= 0xdfff {
				if window.platform.highSurrogate != 0 {
					codepoint += (rune(window.platform.highSurrogate) - 0xd800) << 10
					codepoint += (rune(wParam) & 0xffff) - 0xdc00
					codepoint += 0x10000
				}
			} else {
				codepoint = rune(wParam) & 0xffff
			}

			window.platform.highSurrogate = 0
			window.inputChar(codepoint, getKeyMods(), uMsg != _WM_SYSCHAR)
		}

		return 0

	case _WM_UNICHAR:
		if wParam == _UNICODE_NOCHAR {
			// WM_UNICHAR is not sent by Windows, but is sent by some
			// third-party input method engine
			// Returning TRUE here announces support for this message
			return 1
		}

		window.inputChar(rune(wParam), getKeyMods(), true)
		return 0

	case _WM_KEYDOWN, _WM_SYSKEYDOWN, _WM_KEYUP, _WM_SYSKEYUP:
		action := Press
		if _HIWORD(uint32(lParam))&_KF_UP != 0 {
			action = Release
		}
		mods := getKeyMods()

		scancode := uint32((_HIWORD(uint32(lParam)) & (_KF_EXTENDED | 0xff)))
		if scancode == 0 {
			if microsoftgdk.IsXbox() {
				break
			}
			// NOTE: Some synthetic key messages have a scancode of zero
			// HACK: Map the virtual key back to a usable scancode
			scancode = _MapVirtualKeyW(uint32(wParam), _MAPVK_VK_TO_VSC)
		}

		// HACK: Alt+PrtSc has a different scancode than just PrtSc
		if scancode == 0x54 {
			scancode = 0x137
		}

		// HACK: Ctrl+Pause has a different scancode than just Pause
		if scancode == 0x146 {
			scancode = 0x45
		}

		// HACK: CJK IME sets the extended bit for right Shift
		if scancode == 0x136 {
			scancode = 0x36
		}

		key := _glfw.platformWindow.keycodes[scancode]

		// The Ctrl keys require special handling
		if wParam == _VK_CONTROL {
			if _HIWORD(uint32(lParam))&_KF_EXTENDED != 0 {
				// Right side keys have the extended key bit set
				key = KeyRightControl
			} else {
				// NOTE: Alt Gr sends Left Ctrl followed by Right Alt
				// HACK: We only want one event for Alt Gr, so if we detect
				//       this sequence we discard this Left Ctrl message now
				//       and later report Right Alt normally
				var next _MSG
				time := _GetMessageTime()
				if _PeekMessageW(&next, 0, 0, 0, _PM_NOREMOVE) {
					if next.message == _WM_KEYDOWN ||
						next.message == _WM_SYSKEYDOWN ||
						next.message == _WM_KEYUP ||
						next.message == _WM_SYSKEYUP {
						if next.wParam == _VK_MENU && (_HIWORD(uint32(next.lParam))&_KF_EXTENDED) != 0 && next.time == uint32(time) {
							// Next message is Right Alt down so discard this
							break
						}
					}
				}

				// This is a regular Left Ctrl message
				key = KeyLeftControl
			}
		} else if wParam == _VK_PROCESSKEY {
			// IME notifies that keys have been filtered by setting the
			// virtual key-code to VK_PROCESSKEY
			break
		}

		if action == Release && wParam == _VK_SHIFT {
			// HACK: Release both Shift keys on Shift up event, as when both
			//       are pressed the first release does not emit any event
			// NOTE: The other half of this is in _glfwPlatformPollEvents
			window.inputKey(KeyLeftShift, int(scancode), action, mods)
			window.inputKey(KeyRightShift, int(scancode), action, mods)
		} else if wParam == _VK_SNAPSHOT {
			// HACK: Key down is not reported for the Print Screen key
			window.inputKey(key, int(scancode), Press, mods)
			window.inputKey(key, int(scancode), Release, mods)
		} else {
			window.inputKey(key, int(scancode), action, mods)
		}

	case _WM_LBUTTONDOWN, _WM_RBUTTONDOWN, _WM_MBUTTONDOWN, _WM_XBUTTONDOWN, _WM_LBUTTONUP, _WM_RBUTTONUP, _WM_MBUTTONUP, _WM_XBUTTONUP:
		var button MouseButton
		if uMsg == _WM_LBUTTONDOWN || uMsg == _WM_LBUTTONUP {
			button = MouseButtonLeft
		} else if uMsg == _WM_RBUTTONDOWN || uMsg == _WM_RBUTTONUP {
			button = MouseButtonRight
		} else if uMsg == _WM_MBUTTONDOWN || uMsg == _WM_MBUTTONUP {
			button = MouseButtonMiddle
		} else if _GET_XBUTTON_WPARAM(wParam) == _XBUTTON1 {
			button = MouseButton4
		} else {
			button = MouseButton5
		}

		var action Action
		if uMsg == _WM_LBUTTONDOWN || uMsg == _WM_RBUTTONDOWN || uMsg == _WM_MBUTTONDOWN || uMsg == _WM_XBUTTONDOWN {
			action = Press
		} else {
			action = Release
		}

		var i MouseButton
		for i = 0; i <= MouseButtonLast; i++ {
			if window.mouseButtons[i] == Press {
				break
			}
		}
		if i > MouseButtonLast {
			_SetCapture(hWnd)
		}

		window.inputMouseClick(button, action, getKeyMods())

		for i = 0; i <= MouseButtonLast; i++ {
			if window.mouseButtons[i] == Press {
				break
			}
		}
		if i > MouseButtonLast {
			if err := _ReleaseCapture(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		if uMsg == _WM_XBUTTONDOWN || uMsg == _WM_XBUTTONUP {
			return 1
		}
		return 0

	case _WM_MOUSEMOVE:
		x := _GET_X_LPARAM(lParam)
		y := _GET_Y_LPARAM(lParam)

		if !window.platform.cursorTracked {
			var tme _TRACKMOUSEEVENT
			tme.cbSize = uint32(unsafe.Sizeof(tme))
			tme.dwFlags = _TME_LEAVE
			tme.hwndTrack = window.platform.handle
			if err := _TrackMouseEvent(&tme); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}

			window.platform.cursorTracked = true
			window.inputCursorEnter(true)
		}

		if window.cursorMode == CursorDisabled {
			dx := x - window.platform.lastCursorPosX
			dy := y - window.platform.lastCursorPosY

			if _glfw.platformWindow.disabledCursorWindow != window {
				break
			}
			if window.rawMouseMotion {
				break
			}

			window.inputCursorPos(window.virtualCursorPosX+float64(dx), window.virtualCursorPosY+float64(dy))
		} else {
			window.inputCursorPos(float64(x), float64(y))
		}

		window.platform.lastCursorPosX = x
		window.platform.lastCursorPosY = y

		return 0

	case _WM_INPUT:
		if _glfw.platformWindow.disabledCursorWindow != window {
			break
		}
		if !window.rawMouseMotion {
			break
		}

		ri := _HRAWINPUT(lParam)
		var size uint32
		if _, err := _GetRawInputData(ri, _RID_INPUT, nil, &size); err != nil {
			_glfw.errors = append(_glfw.errors, err)
			return 0
		}
		if size > uint32(len(_glfw.platformWindow.rawInput)) {
			_glfw.platformWindow.rawInput = make([]byte, size)
		}

		size = uint32(len(_glfw.platformWindow.rawInput))
		if _, err := _GetRawInputData(ri, _RID_INPUT, unsafe.Pointer(&_glfw.platformWindow.rawInput[0]), &size); err != nil {
			_glfw.errors = append(_glfw.errors, err)
			return 0
			// TODO: break?
		}

		var dx, dy int
		data := (*_RAWINPUT)(unsafe.Pointer(&_glfw.platformWindow.rawInput[0]))
		if data.mouse.usFlags&_MOUSE_MOVE_ABSOLUTE != 0 {
			if _glfw.platformWindow.isRemoteSession {
				// Remote Desktop Mode
				// As per https://github.com/Microsoft/DirectXTK/commit/ef56b63f3739381e451f7a5a5bd2c9779d2a7555
				// MOUSE_MOVE_ABSOLUTE is a range from 0 through 65535, based on the screen size.
				// Apparently, absolute mode only occurs over RDP though.
				var smx int32 = _SM_CXSCREEN
				var smy int32 = _SM_CYSCREEN
				if data.mouse.usFlags&_MOUSE_VIRTUAL_DESKTOP != 0 {
					smx = _SM_CXVIRTUALSCREEN
					smy = _SM_CYVIRTUALSCREEN
				}

				width, err := _GetSystemMetrics(smx)
				if err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}
				height, err := _GetSystemMetrics(smy)
				if err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}

				pos := _POINT{
					x: int32(float64(data.mouse.lLastX) / 65535.0 * float64(width)),
					y: int32(float64(data.mouse.lLastY) / 65535.0 * float64(height)),
				}
				if err := _ScreenToClient(window.platform.handle, &pos); err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}

				dx = int(pos.x) - window.platform.lastCursorPosX
				dy = int(pos.y) - window.platform.lastCursorPosY
			} else {
				// Normal mode
				// We should have the right absolute coords in data.mouse
				dx = int(data.mouse.lLastX) - window.platform.lastCursorPosX
				dy = int(data.mouse.lLastY) - window.platform.lastCursorPosY
			}
		} else {
			dx = int(data.mouse.lLastX)
			dy = int(data.mouse.lLastY)
		}

		window.inputCursorPos(window.virtualCursorPosX+float64(dx), window.virtualCursorPosY+float64(dy))

		window.platform.lastCursorPosX += dx
		window.platform.lastCursorPosY += dy

	case _WM_MOUSELEAVE:
		window.platform.cursorTracked = false
		window.inputCursorEnter(false)
		return 0

	case _WM_MOUSEWHEEL:
		window.inputScroll(0, float64(int16(_HIWORD(uint32(wParam))))/_WHEEL_DELTA)
		return 0

	case _WM_MOUSEHWHEEL:
		// This message is only sent on Windows Vista and later
		// NOTE: The X-axis is inverted for consistency with macOS and X11
		window.inputScroll(float64(-(int16(_HIWORD(uint32(wParam))))/_WHEEL_DELTA), 0)
		return 0

	case _WM_ENTERSIZEMOVE, _WM_ENTERMENULOOP:
		if window.platform.frameAction {
			break
		}

		// HACK: Enable the cursor while the user is moving or
		//       resizing the window or using the window menu
		if window.cursorMode == CursorDisabled {
			if err := window.enableCursor(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

	case _WM_EXITSIZEMOVE, _WM_EXITMENULOOP:
		if window.platform.frameAction {
			break
		}

		// HACK: Disable the cursor once the user is done moving or
		//       resizing the window or using the menu
		if window.cursorMode == CursorDisabled {
			if err := window.disableCursor(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

	case _WM_SIZE:
		width := int(_LOWORD(uint32(lParam)))
		height := int(_HIWORD(uint32(lParam)))
		iconified := wParam == _SIZE_MINIMIZED
		maximized := wParam == _SIZE_MAXIMIZED || (window.platform.maximized && wParam != _SIZE_RESTORED)

		if _glfw.platformWindow.capturedCursorWindow == window {
			if err := captureCursor(window); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		if window.platform.iconified != iconified {
			window.inputWindowIconify(iconified)
		}

		if window.platform.maximized != maximized {
			window.inputWindowMaximize(maximized)
		}

		if width != window.platform.width || height != window.platform.height {
			window.platform.width = width
			window.platform.height = height

			window.inputFramebufferSize(width, height)
			window.inputWindowSize(width, height)
		}

		if window.monitor != nil && window.platform.iconified != iconified {
			if iconified {
				if err := window.releaseMonitor(); err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}
			} else {
				if err := window.acquireMonitor(); err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}
				if err := window.fitToMonitor(); err != nil {
					_glfw.errors = append(_glfw.errors, err)
					return 0
				}
			}
		}

		window.platform.iconified = iconified
		window.platform.maximized = maximized
		return 0

	case _WM_MOVE:
		if _glfw.platformWindow.capturedCursorWindow == window {
			if err := captureCursor(window); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		// NOTE: This cannot use LOWORD/HIWORD recommended by MSDN, as
		// those macros do not handle negative window positions correctly
		window.inputWindowPos(_GET_X_LPARAM(lParam), _GET_Y_LPARAM(lParam))
		return 0

	case _WM_SIZING:
		if window.numer == DontCare || window.denom == DontCare {
			break
		}

		if err := window.applyAspectRatio(int(wParam), (*_RECT)(unsafe.Pointer(lParam))); err != nil {
			_glfw.errors = append(_glfw.errors, err)
			return 0
		}
		return 1

	case _WM_GETMINMAXINFO:
		var frame _RECT
		mmi := (*_MINMAXINFO)(unsafe.Pointer(lParam))
		style := window.getWindowStyle()
		exStyle := window.getWindowExStyle()

		if window.monitor != nil {
			break
		}

		if winver.IsWindows10AnniversaryUpdateOrGreater() {
			if err := _AdjustWindowRectExForDpi(&frame, style, false, exStyle, _GetDpiForWindow(window.platform.handle)); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		} else {
			if err := _AdjustWindowRectEx(&frame, style, false, exStyle); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		if window.minwidth != DontCare && window.minheight != DontCare {
			mmi.ptMinTrackSize.x = int32(window.minwidth) + (frame.right - frame.left)
			mmi.ptMinTrackSize.y = int32(window.minheight) + (frame.bottom - frame.top)
		}

		if window.maxwidth != DontCare && window.maxheight != DontCare {
			mmi.ptMaxTrackSize.x = int32(window.maxwidth) + (frame.right - frame.left)
			mmi.ptMaxTrackSize.y = int32(window.maxheight) + (frame.bottom - frame.top)
		}

		if !window.decorated {
			mh := _MonitorFromWindow(window.platform.handle, _MONITOR_DEFAULTTONEAREST)
			mi, _ := _GetMonitorInfoW(mh)

			mmi.ptMaxPosition.x = mi.rcWork.left - mi.rcMonitor.left
			mmi.ptMaxPosition.y = mi.rcWork.top - mi.rcMonitor.top
			mmi.ptMaxSize.x = mi.rcWork.right - mi.rcWork.left
			mmi.ptMaxSize.y = mi.rcWork.bottom - mi.rcWork.top
		}

		return 0

	case _WM_PAINT:
		window.inputWindowDamage()

	case _WM_ERASEBKGND:
		return 1

	case _WM_NCACTIVATE, _WM_NCPAINT:
		// Prevent title bar from being drawn after restoring a minimized
		// undecorated window
		if !window.decorated {
			return 1
		}

	case _WM_DWMCOMPOSITIONCHANGED, _WM_DWMCOLORIZATIONCOLORCHANGED:
		if window.platform.transparent {
			if err := window.updateFramebufferTransparency(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}
		return 0

	case _WM_GETDPISCALEDSIZE:
		if window.platform.scaleToMonitor {
			break
		}

		// Adjust the window size to keep the content area size constant
		if winver.IsWindows10CreatorsUpdateOrGreater() {
			var source, target _RECT
			size := (*_SIZE)(unsafe.Pointer(lParam))

			if err := _AdjustWindowRectExForDpi(&source, window.getWindowStyle(), false, window.getWindowExStyle(), _GetDpiForWindow(window.platform.handle)); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
			if err := _AdjustWindowRectExForDpi(&target, window.getWindowStyle(), false, window.getWindowExStyle(), uint32(_LOWORD(uint32(wParam)))); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}

			size.cx += (target.right - target.left) - (source.right - source.left)
			size.cy += (target.bottom - target.top) - (source.bottom - source.top)
			return 1
		}

	case _WM_DPICHANGED:
		xscale := float32(_HIWORD(uint32(wParam))) / float32(_USER_DEFAULT_SCREEN_DPI)
		yscale := float32(_LOWORD(uint32(wParam))) / float32(_USER_DEFAULT_SCREEN_DPI)

		// Resize windowed mode windows that either permit rescaling or that
		// need it to compensate for non-client area scaling
		if window.monitor == nil && (window.platform.scaleToMonitor || winver.IsWindows10CreatorsUpdateOrGreater()) {
			suggested := (*_RECT)(unsafe.Pointer(lParam))
			if err := _SetWindowPos(window.platform.handle, _HWND_TOP,
				suggested.left,
				suggested.top,
				suggested.right-suggested.left,
				suggested.bottom-suggested.top,
				_SWP_NOACTIVATE|_SWP_NOZORDER); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
		}

		window.inputWindowContentScale(xscale, yscale)

	case _WM_SETCURSOR:
		if _LOWORD(uint32(lParam)) == _HTCLIENT {
			if err := window.updateCursorImage(); err != nil {
				_glfw.errors = append(_glfw.errors, err)
				return 0
			}
			return 1
		}

	case _WM_DROPFILES:
		drop := _HDROP(wParam)

		count := _DragQueryFileW(drop, 0xffffffff, nil)
		paths := make([]string, count)

		// Move the mouse to the position of the drop
		pt, _ := _DragQueryPoint(drop)
		window.inputCursorPos(float64(pt.x), float64(pt.y))

		for i := range paths {
			length := _DragQueryFileW(drop, uint32(i), nil)
			buffer := make([]uint16, length+1)
			_DragQueryFileW(drop, uint32(i), buffer)
			paths[i] = windows.UTF16ToString(buffer)
		}

		window.inputDrop(paths)

		_DragFinish(drop)
		return 0
	}

	return uintptr(_DefWindowProcW(hWnd, uMsg, wParam, lParam))
}

var windowProcPtr = windows.NewCallbackCDecl(windowProc)

var handleToWindow = map[windows.HWND]*Window{}

func (w *Window) createNativeWindow(wndconfig *wndconfig, fbconfig *fbconfig) error {
	style := w.getWindowStyle()
	exStyle := w.getWindowExStyle()

	var frameX, frameY, frameWidth, frameHeight int32
	if w.monitor != nil {
		mi, ok := _GetMonitorInfoW(w.monitor.platform.handle)
		if !ok {
			return fmt.Errorf("glfw: GetMonitorInfoW failed")
		}
		// NOTE: This window placement is temporary and approximate, as the
		//       correct position and size cannot be known until the monitor
		//       video mode has been picked in _glfwSetVideoModeWin32
		frameX = mi.rcMonitor.left
		frameY = mi.rcMonitor.top
		frameWidth = mi.rcMonitor.right - mi.rcMonitor.left
		frameHeight = mi.rcMonitor.bottom - mi.rcMonitor.top
	} else {
		rect := _RECT{0, 0, int32(wndconfig.width), int32(wndconfig.height)}

		w.platform.maximized = wndconfig.maximized
		if wndconfig.maximized {
			style |= _WS_MAXIMIZE
		}

		if err := _AdjustWindowRectEx(&rect, style, false, exStyle); err != nil {
			return err
		}

		frameX = _CW_USEDEFAULT
		frameY = _CW_USEDEFAULT
		frameWidth = rect.right - rect.left
		frameHeight = rect.bottom - rect.top
	}

	h, err := _CreateWindowExW(exStyle, _GLFW_WNDCLASSNAME, wndconfig.title, style, frameX, frameY, frameWidth, frameHeight,
		0, // No parent window
		0, // No window menu
		_glfw.platformWindow.instance, unsafe.Pointer(wndconfig))
	if err != nil {
		return err
	}
	w.platform.handle = h

	handleToWindow[w.platform.handle] = w

	if !microsoftgdk.IsXbox() && winver.IsWindows7OrGreater() {
		if err := _ChangeWindowMessageFilterEx(w.platform.handle, _WM_DROPFILES, _MSGFLT_ALLOW, nil); err != nil {
			return err
		}
		if err := _ChangeWindowMessageFilterEx(w.platform.handle, _WM_COPYDATA, _MSGFLT_ALLOW, nil); err != nil {
			return err
		}
		if err := _ChangeWindowMessageFilterEx(w.platform.handle, _WM_COPYGLOBALDATA, _MSGFLT_ALLOW, nil); err != nil {
			return err
		}
	}

	w.platform.scaleToMonitor = wndconfig.scaleToMonitor

	// Adjust window rect to account for DPI scaling of the window frame and
	// (if enabled) DPI scaling of the content area
	// This cannot be done until we know what monitor the window was placed on
	if !microsoftgdk.IsXbox() && w.monitor == nil {
		rect := _RECT{
			left:   0,
			top:    0,
			right:  int32(wndconfig.width),
			bottom: int32(wndconfig.height),
		}
		mh := _MonitorFromWindow(w.platform.handle, _MONITOR_DEFAULTTONEAREST)

		// Adjust window rect to account for DPI scaling of the window frame and
		// (if enabled) DPI scaling of the content area
		// This cannot be done until we know what monitor the window was placed on
		// Only update the restored window rect as the window may be maximized

		if wndconfig.scaleToMonitor {
			xscale, yscale, err := getMonitorContentScaleWin32(mh)
			if err != nil {
				return err
			}
			if xscale > 0 && yscale > 0 {
				rect.right = int32(float32(rect.right) * xscale)
				rect.bottom = int32(float32(rect.bottom) * yscale)
			}
		}

		rect, err = w.clientToScreen(rect)
		if err != nil {
			return err
		}

		if winver.IsWindows10AnniversaryUpdateOrGreater() {
			if err := _AdjustWindowRectExForDpi(&rect, style, false, exStyle, _GetDpiForWindow(w.platform.handle)); err != nil {
				return err
			}
		} else {
			if err := _AdjustWindowRectEx(&rect, style, false, exStyle); err != nil {
				return err
			}
		}

		// Only update the restored window rect as the window may be maximized
		wp, err := _GetWindowPlacement(w.platform.handle)
		if err != nil {
			return err
		}
		_OffsetRect(&rect, wp.rcNormalPosition.left-rect.left, wp.rcNormalPosition.top-rect.top)

		wp.rcNormalPosition = rect
		wp.showCmd = _SW_HIDE
		if err := _SetWindowPlacement(w.platform.handle, &wp); err != nil {
			return err
		}

		// Adjust rect of maximized undecorated window, because by default Windows will
		// make such a window cover the whole monitor instead of its workarea

		if wndconfig.maximized && !wndconfig.decorated {
			mi, _ := _GetMonitorInfoW(mh)
			if err := _SetWindowPos(w.platform.handle, _HWND_TOP,
				mi.rcWork.left, mi.rcWork.top, mi.rcWork.right-mi.rcWork.left, mi.rcWork.bottom-mi.rcWork.top,
				_SWP_NOACTIVATE|_SWP_NOZORDER); err != nil {
				return err
			}
		}
	}

	if !microsoftgdk.IsXbox() {
		_DragAcceptFiles(w.platform.handle, true)
	}

	if fbconfig.transparent {
		if err := w.updateFramebufferTransparency(); err != nil {
			return err
		}
		w.platform.transparent = true
	}

	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return err
	}
	w.platform.width, w.platform.height = width, height

	return nil
}

func registerWindowClassWin32() error {
	var wc _WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.style = _CS_HREDRAW | _CS_VREDRAW | _CS_OWNDC
	wc.lpfnWndProc = _WNDPROC(windowProcPtr)
	wc.hInstance = _glfw.platformWindow.instance
	cursor, err := _LoadCursorW(0, _IDC_ARROW)
	if err != nil {
		return err
	}
	wc.hCursor = cursor
	className, err := windows.UTF16FromString(_GLFW_WNDCLASSNAME)
	if err != nil {
		panic("glfw: _GLFW_WNDCLASSNAME must not include a NUL character")
	}
	wc.lpszClassName = &className[0]
	defer runtime.KeepAlive(className)

	// In the original GLFW implementation, an embedded resource GLFW_ICON is used if possible.
	// See https://www.glfw.org/docs/3.3/group__window.html

	if !microsoftgdk.IsXbox() {
		icon, err := _LoadImageW(0, _IDI_APPLICATION, _IMAGE_ICON, 0, 0, _LR_DEFAULTSIZE|_LR_SHARED)
		if err != nil {
			return err
		}
		wc.hIcon = _HICON(icon)
	}

	if _, err := _RegisterClassExW(&wc); err != nil {
		return err
	}
	return nil
}

func unregisterWindowClassWin32() error {
	if err := _UnregisterClassW(_GLFW_WNDCLASSNAME, _glfw.platformWindow.instance); err != nil {
		return err
	}
	return nil
}

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	if err := w.createNativeWindow(wndconfig, fbconfig); err != nil {
		return err
	}

	if ctxconfig.client != NoAPI {
		if ctxconfig.source == NativeContextAPI {
			if err := initWGL(); err != nil {
				return err
			}
			if err := w.createContextWGL(ctxconfig, fbconfig); err != nil {
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
		if err := w.platformFocusWindow(); err != nil {
			return err
		}
		if err := w.acquireMonitor(); err != nil {
			return err
		}
		if err := w.fitToMonitor(); err != nil {
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

	return nil
}

func (w *Window) platformDestroyWindow() error {
	if w.monitor != nil {
		if err := w.releaseMonitor(); err != nil {
			return err
		}
	}

	if w.context.destroy != nil {
		if err := w.context.destroy(w); err != nil {
			return err
		}
	}

	if _glfw.platformWindow.disabledCursorWindow == w {
		if err := w.enableCursor(); err != nil {
			return err
		}
	}

	if _glfw.platformWindow.capturedCursorWindow == w {
		if err := releaseCursor(); err != nil {
			return err
		}
	}

	if w.platform.handle != 0 {
		if !microsoftgdk.IsXbox() {
			// An error 'invalid window handle' can occur without any specific reasons (#2551).
			// As there is nothing to do, just ignore this error.
			if err := _DestroyWindow(w.platform.handle); err != nil && !errors.Is(err, windows.ERROR_INVALID_WINDOW_HANDLE) {
				return err
			}
		}
		delete(handleToWindow, w.platform.handle)
		w.platform.handle = 0
	}

	if w.platform.bigIcon != 0 {
		if err := _DestroyIcon(w.platform.bigIcon); err != nil {
			return err
		}
	}

	if w.platform.smallIcon != 0 {
		if err := _DestroyIcon(w.platform.smallIcon); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) platformSetWindowTitle(title string) error {
	if microsoftgdk.IsXbox() {
		return nil
	}
	return _SetWindowTextW(w.platform.handle, title)
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	var bigIcon, smallIcon _HICON

	if len(images) > 0 {
		cxIcon, err := _GetSystemMetrics(_SM_CXICON)
		if err != nil {
			return err
		}
		cyIcon, err := _GetSystemMetrics(_SM_CYICON)
		if err != nil {
			return err
		}
		cxsmIcon, err := _GetSystemMetrics(_SM_CXSMICON)
		if err != nil {
			return err
		}
		cysmIcon, err := _GetSystemMetrics(_SM_CYSMICON)
		if err != nil {
			return err
		}

		bigImage := chooseImage(images, int(cxIcon), int(cyIcon))
		smallImage := chooseImage(images, int(cxsmIcon), int(cysmIcon))

		bigIcon, err = createIcon(bigImage, 0, 0, true)
		if err != nil {
			return err
		}
		smallIcon, err = createIcon(smallImage, 0, 0, false)
		if err != nil {
			return err
		}
	} else {
		i, err := _GetClassLongPtrW(w.platform.handle, _GCLP_HICON)
		if err != nil {
			return err
		}
		bigIcon = _HICON(i)
		i, err = _GetClassLongPtrW(w.platform.handle, _GCLP_HICONSM)
		if err != nil {
			return err
		}
		smallIcon = _HICON(i)
	}

	_SendMessageW(w.platform.handle, _WM_SETICON, _ICON_BIG, _LPARAM(bigIcon))
	_SendMessageW(w.platform.handle, _WM_SETICON, _ICON_SMALL, _LPARAM(smallIcon))

	if w.platform.bigIcon != 0 {
		if err := _DestroyIcon(w.platform.bigIcon); err != nil {
			return err
		}
	}

	if w.platform.smallIcon != 0 {
		if err := _DestroyIcon(w.platform.smallIcon); err != nil {
			return err
		}
	}

	if len(images) > 0 {
		w.platform.bigIcon = bigIcon
		w.platform.smallIcon = smallIcon
	} else {
		w.platform.bigIcon = 0
		w.platform.smallIcon = 0
	}
	return nil
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	if microsoftgdk.IsXbox() {
		return 0, 0, nil
	}

	var pos _POINT
	if err := _ClientToScreen(w.platform.handle, &pos); err != nil {
		return 0, 0, err
	}
	return int(pos.x), int(pos.y), nil
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	rect := _RECT{
		left:   int32(xpos),
		top:    int32(ypos),
		right:  int32(xpos),
		bottom: int32(ypos),
	}
	if winver.IsWindows10AnniversaryUpdateOrGreater() {
		if err := _AdjustWindowRectExForDpi(&rect, w.getWindowStyle(), false, w.getWindowExStyle(), _GetDpiForWindow(w.platform.handle)); err != nil {
			return err
		}
	} else {
		if err := _AdjustWindowRectEx(&rect, w.getWindowStyle(), false, w.getWindowExStyle()); err != nil {
			return err
		}
	}

	if err := _SetWindowPos(w.platform.handle, 0, rect.left, rect.top, 0, 0, _SWP_NOACTIVATE|_SWP_NOZORDER|_SWP_NOSIZE); err != nil {
		return err
	}
	return nil
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	area, err := _GetClientRect(w.platform.handle)
	if err != nil {
		return 0, 0, err
	}
	return int(area.right), int(area.bottom), nil
}

func (w *Window) platformSetWindowSize(width, height int) error {
	if w.monitor != nil {
		if w.monitor.window == w {
			if err := w.acquireMonitor(); err != nil {
				return err
			}
			if err := w.fitToMonitor(); err != nil {
				return err
			}
		}
	} else {
		rect := _RECT{
			left:   0,
			top:    0,
			right:  int32(width),
			bottom: int32(height),
		}

		if winver.IsWindows10AnniversaryUpdateOrGreater() {
			if err := _AdjustWindowRectExForDpi(&rect, w.getWindowStyle(), false, w.getWindowExStyle(), _GetDpiForWindow(w.platform.handle)); err != nil {
				return err
			}
		} else {
			if err := _AdjustWindowRectEx(&rect, w.getWindowStyle(), false, w.getWindowExStyle()); err != nil {
				return err
			}
		}

		if err := _SetWindowPos(w.platform.handle, _HWND_TOP,
			0, 0, rect.right-rect.left, rect.bottom-rect.top,
			_SWP_NOACTIVATE|_SWP_NOOWNERZORDER|_SWP_NOMOVE|_SWP_NOZORDER); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	if (minwidth == DontCare || minheight == DontCare) && (maxwidth == DontCare || maxheight == DontCare) {
		return nil
	}

	area, err := _GetWindowRect(w.platform.handle)
	if err != nil {
		return err
	}
	if err := _MoveWindow(w.platform.handle, area.left, area.top, area.right-area.left, area.bottom-area.top, true); err != nil {
		return err
	}

	if err := w.updateWindowStyles(); err != nil {
		return err
	}
	return nil
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	if numer == DontCare || denom == DontCare {
		return nil
	}

	area, err := _GetWindowRect(w.platform.handle)
	if err != nil {
		return err
	}
	if err := w.applyAspectRatio(_WMSZ_BOTTOMRIGHT, &area); err != nil {
		return err
	}
	if err := _MoveWindow(w.platform.handle, area.left, area.top, area.right-area.left, area.bottom-area.top, true); err != nil {
		return err
	}
	return nil
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	return w.platformGetWindowSize()
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	rect := _RECT{
		left:   0,
		top:    0,
		right:  int32(width),
		bottom: int32(height),
	}
	if winver.IsWindows10AnniversaryUpdateOrGreater() {
		if err := _AdjustWindowRectExForDpi(&rect, w.getWindowStyle(), false, w.getWindowExStyle(), _GetDpiForWindow(w.platform.handle)); err != nil {
			return 0, 0, 0, 0, err
		}
	} else {
		if err := _AdjustWindowRectEx(&rect, w.getWindowStyle(), false, w.getWindowExStyle()); err != nil {
			return 0, 0, 0, 0, err
		}
	}

	return -int(rect.left), -int(rect.top), int(rect.right) - width, int(rect.bottom) - height, nil
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	handle := _MonitorFromWindow(w.platform.handle, _MONITOR_DEFAULTTONEAREST)
	return getMonitorContentScaleWin32(handle)
}

func (w *Window) platformIconifyWindow() {
	_ShowWindow(w.platform.handle, _SW_MINIMIZE)
}

func (w *Window) platformRestoreWindow() {
	_ShowWindow(w.platform.handle, _SW_RESTORE)
}

func (w *Window) platformMaximizeWindow() error {
	if _IsWindowVisible(w.platform.handle) {
		_ShowWindow(w.platform.handle, _SW_MAXIMIZE)
	} else {
		if err := w.maximizeWindowManually(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Window) platformShowWindow() {
	_ShowWindow(w.platform.handle, _SW_SHOWNA)
}

func (w *Window) platformHideWindow() {
	_ShowWindow(w.platform.handle, _SW_HIDE)
}

func (w *Window) platformRequestWindowAttention() {
	_FlashWindow(w.platform.handle, true)
}

func (w *Window) platformFocusWindow() error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	if err := _BringWindowToTop(w.platform.handle); err != nil {
		return err
	}
	_SetForegroundWindow(w.platform.handle)
	if _, err := _SetFocus(w.platform.handle); err != nil {
		return err
	}
	return nil
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	if w.monitor == monitor {
		if monitor != nil {
			if monitor.window == w {
				if err := w.acquireMonitor(); err != nil {
					return err
				}
				if err := w.fitToMonitor(); err != nil {
					return err
				}
			}
		} else {
			rect := _RECT{
				left:   int32(xpos),
				top:    int32(ypos),
				right:  int32(xpos + width),
				bottom: int32(ypos + height),
			}
			if winver.IsWindows10AnniversaryUpdateOrGreater() {
				if err := _AdjustWindowRectExForDpi(&rect, w.getWindowStyle(), false, w.getWindowExStyle(), _GetDpiForWindow(w.platform.handle)); err != nil {
					return err
				}
			} else {
				if err := _AdjustWindowRectEx(&rect, w.getWindowStyle(), false, w.getWindowExStyle()); err != nil {
					return err
				}
			}

			if err := _SetWindowPos(w.platform.handle, _HWND_TOP,
				rect.left, rect.top, rect.right-rect.left, rect.bottom-rect.top,
				_SWP_NOCOPYBITS|_SWP_NOACTIVATE|_SWP_NOZORDER); err != nil {
				return err
			}
		}

		return nil
	}

	if w.monitor != nil {
		if err := w.releaseMonitor(); err != nil {
			return err
		}
	}

	w.inputWindowMonitor(monitor)

	if w.monitor != nil {
		var flags uint32 = _SWP_SHOWWINDOW | _SWP_NOACTIVATE | _SWP_NOCOPYBITS
		if w.decorated {
			s, err := _GetWindowLongW(w.platform.handle, _GWL_STYLE)
			if err != nil {
				return err
			}
			style := uint32(s)
			style &^= _WS_OVERLAPPEDWINDOW
			style |= w.getWindowStyle()
			if _, err := _SetWindowLongW(w.platform.handle, _GWL_STYLE, int32(style)); err != nil {
				return err
			}
			flags |= _SWP_FRAMECHANGED
		}

		if err := w.acquireMonitor(); err != nil {
			return err
		}

		mi, _ := _GetMonitorInfoW(w.monitor.platform.handle)
		var hWnd windows.HWND = _HWND_NOTOPMOST
		if w.floating {
			hWnd = _HWND_TOPMOST
		}
		if err := _SetWindowPos(w.platform.handle, hWnd,
			mi.rcMonitor.left,
			mi.rcMonitor.top,
			mi.rcMonitor.right-mi.rcMonitor.left,
			mi.rcMonitor.bottom-mi.rcMonitor.top,
			flags); err != nil {
			return err
		}
	} else {
		var flags uint32 = _SWP_NOACTIVATE | _SWP_NOCOPYBITS
		if w.decorated {
			s, err := _GetWindowLongW(w.platform.handle, _GWL_STYLE)
			if err != nil {
				return err
			}
			style := uint32(s)
			style &^= _WS_POPUP
			style |= w.getWindowStyle()
			if _, err := _SetWindowLongW(w.platform.handle, _GWL_STYLE, int32(style)); err != nil {
				return err
			}
			flags |= _SWP_FRAMECHANGED
		}

		rect := _RECT{
			left:   int32(xpos),
			top:    int32(ypos),
			right:  int32(xpos + width),
			bottom: int32(ypos + height),
		}
		if winver.IsWindows10AnniversaryUpdateOrGreater() {
			if err := _AdjustWindowRectExForDpi(&rect, w.getWindowStyle(),
				false, w.getWindowExStyle(),
				_GetDpiForWindow(w.platform.handle)); err != nil {
				return err
			}
		} else {
			if err := _AdjustWindowRectEx(&rect, w.getWindowStyle(),
				false, w.getWindowExStyle()); err != nil {
				return err
			}
		}

		var after windows.HWND
		if w.floating {
			after = _HWND_TOPMOST
		} else {
			after = _HWND_NOTOPMOST
		}
		if err := _SetWindowPos(w.platform.handle, after,
			rect.left, rect.top, rect.right-rect.left, rect.bottom-rect.top,
			flags); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) platformWindowFocused() bool {
	if microsoftgdk.IsXbox() {
		return true
	}
	return w.platform.handle == _GetActiveWindow()
}

func (w *Window) platformWindowIconified() bool {
	if microsoftgdk.IsXbox() {
		return false
	}
	return _IsIconic(w.platform.handle)
}

func (w *Window) platformWindowVisible() bool {
	if microsoftgdk.IsXbox() {
		return true
	}
	return _IsWindowVisible(w.platform.handle)
}

func (w *Window) platformWindowMaximized() bool {
	if microsoftgdk.IsXbox() {
		return false
	}
	return _IsZoomed(w.platform.handle)
}

func (w *Window) platformWindowHovered() (bool, error) {
	if microsoftgdk.IsXbox() {
		return true, nil
	}
	return w.cursorInContentArea()
}

func (w *Window) platformFramebufferTransparent() bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	if !w.platform.transparent {
		return false
	}

	if !winver.IsWindowsVistaOrGreater() {
		return false
	}

	composition, err := _DwmIsCompositionEnabled()
	if err != nil || !composition {
		return false
	}

	if !winver.IsWindows8OrGreater() {
		// HACK: Disable framebuffer transparency on Windows 7 when the
		//       colorization color is opaque, because otherwise the window
		//       contents is blended additively with the previous frame instead
		//       of replacing it
		_, opaque, err := _DwmGetColorizationColor()
		if err != nil || opaque {
			return false
		}
	}

	return true
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	return w.updateWindowStyles()
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	return w.updateWindowStyles()
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	var after windows.HWND = _HWND_NOTOPMOST
	if enabled {
		after = _HWND_TOPMOST
	}
	return _SetWindowPos(w.platform.handle, after, 0, 0, 0, 0, _SWP_NOACTIVATE|_SWP_NOMOVE|_SWP_NOSIZE)
}

func (w *Window) platformSetWindowMousePassthrough(enabled bool) error {
	exStyle, err := _GetWindowLongW(w.platform.handle, _GWL_EXSTYLE)
	if err != nil {
		return err
	}

	var key _COLORREF
	var alpha byte
	var flags uint32
	if exStyle&_WS_EX_LAYERED != 0 {
		var err error
		key, alpha, flags, err = _GetLayeredWindowAttributes(w.platform.handle)
		if err != nil {
			return err
		}
	}

	if enabled {
		exStyle |= _WS_EX_TRANSPARENT | _WS_EX_LAYERED
	} else {
		exStyle &^= _WS_EX_TRANSPARENT
		// NOTE: Window opacity also needs the layered window style so do not
		//       remove it if the window is alpha blended
		if exStyle&_WS_EX_LAYERED != 0 {
			if flags&_LWA_ALPHA == 0 {
				exStyle &^= _WS_EX_LAYERED
			}
		}
	}

	if _, err := _SetWindowLongW(w.platform.handle, _GWL_EXSTYLE, exStyle); err != nil {
		return err
	}

	if enabled {
		if err := _SetLayeredWindowAttributes(w.platform.handle, key, alpha, flags); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	style, err := _GetWindowLongW(w.platform.handle, _GWL_EXSTYLE)
	if err != nil {
		return 0, err
	}

	if style&_WS_EX_LAYERED != 0 {
		_, alpha, flags, err := _GetLayeredWindowAttributes(w.platform.handle)
		if err != nil {
			return 0, err
		}

		if flags&_LWA_ALPHA != 0 {
			return float32(alpha) / 255, nil
		}
	}

	return 1, nil
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	if opacity < 1 {
		alpha := byte(255 * opacity)
		style, err := _GetWindowLongW(w.platform.handle, _GWL_EXSTYLE)
		if err != nil {
			return err
		}
		style |= _WS_EX_LAYERED
		if _, err := _SetWindowLongW(w.platform.handle, _GWL_EXSTYLE, style); err != nil {
			return err
		}
		if err := _SetLayeredWindowAttributes(w.platform.handle, 0, alpha, _LWA_ALPHA); err != nil {
			return err
		}
	} else {
		style, err := _GetWindowLongW(w.platform.handle, _GWL_EXSTYLE)
		if err != nil {
			return err
		}
		style &^= _WS_EX_LAYERED
		if _, err := _SetWindowLongW(w.platform.handle, _GWL_EXSTYLE, style); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	if _glfw.platformWindow.disabledCursorWindow != w {
		return nil
	}

	if enabled {
		if err := w.enableRawMouseMotion(); err != nil {
			return err
		}
	} else {
		if err := w.disableRawMouseMotion(); err != nil {
			return err
		}
	}
	return nil
}

func platformRawMouseMotionSupported() bool {
	return true
}

func platformPollEvents() error {
	if len(_glfw.errors) > 0 {
		return _glfw.errors[0]
	}

	var msg _MSG
	for _PeekMessageW(&msg, 0, 0, 0, _PM_REMOVE) {
		if msg.message == _WM_QUIT {
			// NOTE: While GLFW does not itself post WM_QUIT, other processes
			//       may post it to this one, for example Task Manager
			// HACK: Treat WM_QUIT as a close on all windows
			for _, window := range _glfw.windows {
				window.inputWindowCloseRequest()
			}
		} else {
			_TranslateMessage(&msg)
			_DispatchMessageW(&msg)
		}
	}

	var handle windows.HWND
	if microsoftgdk.IsXbox() {
		// Assume that there is always exactly one active window.
		handle = _glfw.windows[0].platform.handle
	} else {
		handle = _GetActiveWindow()
	}

	// HACK: Release modifier keys that the system did not emit KEYUP for
	// NOTE: Shift keys on Windows tend to "stick" when both are pressed as
	//       no key up message is generated by the first key release
	// NOTE: Windows key is not reported as released by the Win+V hotkey
	//       Other Win hotkeys are handled implicitly by _glfwInputWindowFocus
	//       because they change the input focus
	// NOTE: The other half of this is in the WM_*KEY* handler in windowProc
	if handle != 0 {
		if window := handleToWindow[handle]; window != nil {
			keys := [...]struct {
				VK  int
				Key Key
			}{
				{_VK_LSHIFT, KeyLeftShift},
				{_VK_RSHIFT, KeyRightShift},
				{_VK_LWIN, KeyLeftSuper},
				{_VK_RWIN, KeyRightSuper},
			}
			for i := range keys {
				vk := keys[i].VK
				key := keys[i].Key
				scancode := _glfw.platformWindow.scancodes[key]

				if uint32(_GetKeyState(int32(vk)))&0x8000 != 0 {
					continue
				}
				if window.keys[key] != Press {
					continue
				}
				window.inputKey(key, int(scancode), Release, getKeyMods())
			}
		}
	}

	if window := _glfw.platformWindow.disabledCursorWindow; window != nil {
		width, height, err := window.platformGetWindowSize()
		if err != nil {
			return err
		}

		// NOTE: Re-center the cursor only if it has moved since the last call,
		//       to avoid breaking glfwWaitEvents with WM_MOUSEMOVE
		// The re-center is required in order to prevent the mouse cursor stopping at the edges of the screen.
		if window.platform.lastCursorPosX != width/2 || window.platform.lastCursorPosY != height/2 {
			if err := window.platformSetCursorPos(float64(width/2), float64(height/2)); err != nil {
				return err
			}
		}
	}

	return nil
}

func platformWaitEvents() error {
	if err := _WaitMessage(); err != nil {
		return err
	}
	if err := platformPollEvents(); err != nil {
		return err
	}
	return nil
}

func platformWaitEventsTimeout(timeout float64) error {
	if _, err := _MsgWaitForMultipleObjects(0, nil, false, uint32(timeout*1e3), _QS_ALLINPUT); err != nil {
		return err
	}
	if err := platformPollEvents(); err != nil {
		return err
	}
	return nil
}

func platformPostEmptyEvent() error {
	return _PostMessageW(_glfw.platformWindow.helperWindowHandle, _WM_NULL, 0, 0)
}

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	pos, err := _GetCursorPos()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if !microsoftgdk.IsXbox() {
		if err := _ScreenToClient(w.platform.handle, &pos); err != nil {
			return 0, 0, err
		}
	}
	return float64(pos.x), float64(pos.y), nil
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	pos := _POINT{
		x: int32(xpos),
		y: int32(ypos),
	}

	// Store the new position so it can be recognized later
	w.platform.lastCursorPosX = int(pos.x)
	w.platform.lastCursorPosY = int(pos.y)

	if !microsoftgdk.IsXbox() {
		if err := _ClientToScreen(w.platform.handle, &pos); err != nil {
			return err
		}
	}
	if err := _SetCursorPos(pos.x, pos.y); err != nil {
		return err
	}
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
				if err := w.enableRawMouseMotion(); err != nil {
					return err
				}
			}
		} else if _glfw.platformWindow.disabledCursorWindow == w {
			if w.rawMouseMotion {
				if err := w.disableRawMouseMotion(); err != nil {
					return err
				}
			}
		}

		if mode == CursorDisabled {
			if err := captureCursor(w); err != nil {
				return err
			}
		} else {
			if err := releaseCursor(); err != nil {
				return err
			}
		}

		if mode == CursorDisabled {
			_glfw.platformWindow.disabledCursorWindow = w
		} else if _glfw.platformWindow.disabledCursorWindow == w {
			_glfw.platformWindow.disabledCursorWindow = nil
			if err := w.platformSetCursorPos(_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY); err != nil {
				return err
			}
		}
	}

	in, err := w.cursorInContentArea()
	if err != nil {
		return err
	}
	if in {
		if err := w.updateCursorImage(); err != nil {
			return err
		}
	}

	return nil
}

func platformGetScancodeName(scancode int) (string, error) {
	if scancode < 0 || scancode > (_KF_EXTENDED|0xff) {
		return "", fmt.Errorf("glwfwin: invalid scancode %d: %w", scancode, InvalidValue)
	}
	key := _glfw.platformWindow.keycodes[scancode]
	if key == KeyUnknown {
		return "", nil
	}
	return _glfw.platformWindow.keynames[key], nil
}

func platformGetKeyScancode(key Key) int {
	return _glfw.platformWindow.scancodes[key]
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	var id int
	switch shape {
	case ArrowCursor:
		id = _OCR_NORMAL
	case IBeamCursor:
		id = _OCR_IBEAM
	case CrosshairCursor:
		id = _OCR_CROSS
	case HandCursor:
		id = _OCR_HAND
	case HResizeCursor:
		id = _OCR_SIZEWE
	case VResizeCursor:
		id = _OCR_SIZENS
	case ResizeNWSECursor: // v3.4
		id = _OCR_SIZENWSE
	case ResizeNESWCursor: // v3.4
		id = _OCR_SIZENESW
	case ResizeAllCursor: // v3.4
		id = _OCR_SIZEALL
	case NotAllowedCursor: // v3.4
		id = _OCR_NO
	default:
		return fmt.Errorf("glfw: invalid shape: %d", shape)
	}

	h, err := _LoadImageW(0, uintptr(id), _IMAGE_CURSOR, 0, 0, _LR_DEFAULTSIZE|_LR_SHARED)
	if err != nil {
		return err
	}
	c.platform.handle = _HCURSOR(h)

	return nil
}

func (c *Cursor) platformDestroyCursor() error {
	if c.platform.handle != 0 {
		if err := _DestroyIcon(_HICON(c.platform.handle)); err != nil {
			return err
		}
	}
	return nil
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	in, err := w.cursorInContentArea()
	if err != nil {
		return err
	}
	if in {
		if err := w.updateCursorImage(); err != nil {
			return err
		}
	}
	return nil
}

func platformSetClipboardString(str string) error {
	panic("glfw: platformSetClipboardString is not implemented")
}

func platformGetClipboardString() (string, error) {
	panic("glfw: platformGetClipboardString is not implemented")
}

func (w *Window) GetWin32Window() (windows.HWND, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}
	return w.platform.handle, nil
}

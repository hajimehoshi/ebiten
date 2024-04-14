// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

func createKeyTables() {
	for i := range _glfw.platformWindow.keycodes {
		_glfw.platformWindow.keycodes[i] = -1
	}
	for i := range _glfw.platformWindow.scancodes {
		_glfw.platformWindow.keycodes[i] = -1
	}

	_glfw.platformWindow.keycodes[0x00B] = Key0
	_glfw.platformWindow.keycodes[0x002] = Key1
	_glfw.platformWindow.keycodes[0x003] = Key2
	_glfw.platformWindow.keycodes[0x004] = Key3
	_glfw.platformWindow.keycodes[0x005] = Key4
	_glfw.platformWindow.keycodes[0x006] = Key5
	_glfw.platformWindow.keycodes[0x007] = Key6
	_glfw.platformWindow.keycodes[0x008] = Key7
	_glfw.platformWindow.keycodes[0x009] = Key8
	_glfw.platformWindow.keycodes[0x00A] = Key9
	_glfw.platformWindow.keycodes[0x01E] = KeyA
	_glfw.platformWindow.keycodes[0x030] = KeyB
	_glfw.platformWindow.keycodes[0x02E] = KeyC
	_glfw.platformWindow.keycodes[0x020] = KeyD
	_glfw.platformWindow.keycodes[0x012] = KeyE
	_glfw.platformWindow.keycodes[0x021] = KeyF
	_glfw.platformWindow.keycodes[0x022] = KeyG
	_glfw.platformWindow.keycodes[0x023] = KeyH
	_glfw.platformWindow.keycodes[0x017] = KeyI
	_glfw.platformWindow.keycodes[0x024] = KeyJ
	_glfw.platformWindow.keycodes[0x025] = KeyK
	_glfw.platformWindow.keycodes[0x026] = KeyL
	_glfw.platformWindow.keycodes[0x032] = KeyM
	_glfw.platformWindow.keycodes[0x031] = KeyN
	_glfw.platformWindow.keycodes[0x018] = KeyO
	_glfw.platformWindow.keycodes[0x019] = KeyP
	_glfw.platformWindow.keycodes[0x010] = KeyQ
	_glfw.platformWindow.keycodes[0x013] = KeyR
	_glfw.platformWindow.keycodes[0x01F] = KeyS
	_glfw.platformWindow.keycodes[0x014] = KeyT
	_glfw.platformWindow.keycodes[0x016] = KeyU
	_glfw.platformWindow.keycodes[0x02F] = KeyV
	_glfw.platformWindow.keycodes[0x011] = KeyW
	_glfw.platformWindow.keycodes[0x02D] = KeyX
	_glfw.platformWindow.keycodes[0x015] = KeyY
	_glfw.platformWindow.keycodes[0x02C] = KeyZ

	_glfw.platformWindow.keycodes[0x028] = KeyApostrophe
	_glfw.platformWindow.keycodes[0x02B] = KeyBackslash
	_glfw.platformWindow.keycodes[0x033] = KeyComma
	_glfw.platformWindow.keycodes[0x00D] = KeyEqual
	_glfw.platformWindow.keycodes[0x029] = KeyGraveAccent
	_glfw.platformWindow.keycodes[0x01A] = KeyLeftBracket
	_glfw.platformWindow.keycodes[0x00C] = KeyMinus
	_glfw.platformWindow.keycodes[0x034] = KeyPeriod
	_glfw.platformWindow.keycodes[0x01B] = KeyRightBracket
	_glfw.platformWindow.keycodes[0x027] = KeySemicolon
	_glfw.platformWindow.keycodes[0x035] = KeySlash
	_glfw.platformWindow.keycodes[0x056] = KeyWorld2

	_glfw.platformWindow.keycodes[0x00E] = KeyBackspace
	_glfw.platformWindow.keycodes[0x153] = KeyDelete
	_glfw.platformWindow.keycodes[0x14F] = KeyEnd
	_glfw.platformWindow.keycodes[0x01C] = KeyEnter
	_glfw.platformWindow.keycodes[0x001] = KeyEscape
	_glfw.platformWindow.keycodes[0x147] = KeyHome
	_glfw.platformWindow.keycodes[0x152] = KeyInsert
	_glfw.platformWindow.keycodes[0x15D] = KeyMenu
	_glfw.platformWindow.keycodes[0x151] = KeyPageDown
	_glfw.platformWindow.keycodes[0x149] = KeyPageUp
	_glfw.platformWindow.keycodes[0x045] = KeyPause
	_glfw.platformWindow.keycodes[0x039] = KeySpace
	_glfw.platformWindow.keycodes[0x00F] = KeyTab
	_glfw.platformWindow.keycodes[0x03A] = KeyCapsLock
	_glfw.platformWindow.keycodes[0x145] = KeyNumLock
	_glfw.platformWindow.keycodes[0x046] = KeyScrollLock
	_glfw.platformWindow.keycodes[0x03B] = KeyF1
	_glfw.platformWindow.keycodes[0x03C] = KeyF2
	_glfw.platformWindow.keycodes[0x03D] = KeyF3
	_glfw.platformWindow.keycodes[0x03E] = KeyF4
	_glfw.platformWindow.keycodes[0x03F] = KeyF5
	_glfw.platformWindow.keycodes[0x040] = KeyF6
	_glfw.platformWindow.keycodes[0x041] = KeyF7
	_glfw.platformWindow.keycodes[0x042] = KeyF8
	_glfw.platformWindow.keycodes[0x043] = KeyF9
	_glfw.platformWindow.keycodes[0x044] = KeyF10
	_glfw.platformWindow.keycodes[0x057] = KeyF11
	_glfw.platformWindow.keycodes[0x058] = KeyF12
	_glfw.platformWindow.keycodes[0x064] = KeyF13
	_glfw.platformWindow.keycodes[0x065] = KeyF14
	_glfw.platformWindow.keycodes[0x066] = KeyF15
	_glfw.platformWindow.keycodes[0x067] = KeyF16
	_glfw.platformWindow.keycodes[0x068] = KeyF17
	_glfw.platformWindow.keycodes[0x069] = KeyF18
	_glfw.platformWindow.keycodes[0x06A] = KeyF19
	_glfw.platformWindow.keycodes[0x06B] = KeyF20
	_glfw.platformWindow.keycodes[0x06C] = KeyF21
	_glfw.platformWindow.keycodes[0x06D] = KeyF22
	_glfw.platformWindow.keycodes[0x06E] = KeyF23
	_glfw.platformWindow.keycodes[0x076] = KeyF24
	_glfw.platformWindow.keycodes[0x038] = KeyLeftAlt
	_glfw.platformWindow.keycodes[0x01D] = KeyLeftControl
	_glfw.platformWindow.keycodes[0x02A] = KeyLeftShift
	_glfw.platformWindow.keycodes[0x15B] = KeyLeftSuper
	_glfw.platformWindow.keycodes[0x137] = KeyPrintScreen
	_glfw.platformWindow.keycodes[0x138] = KeyRightAlt
	_glfw.platformWindow.keycodes[0x11D] = KeyRightControl
	_glfw.platformWindow.keycodes[0x036] = KeyRightShift
	_glfw.platformWindow.keycodes[0x15C] = KeyRightSuper
	_glfw.platformWindow.keycodes[0x150] = KeyDown
	_glfw.platformWindow.keycodes[0x14B] = KeyLeft
	_glfw.platformWindow.keycodes[0x14D] = KeyRight
	_glfw.platformWindow.keycodes[0x148] = KeyUp

	_glfw.platformWindow.keycodes[0x052] = KeyKP0
	_glfw.platformWindow.keycodes[0x04F] = KeyKP1
	_glfw.platformWindow.keycodes[0x050] = KeyKP2
	_glfw.platformWindow.keycodes[0x051] = KeyKP3
	_glfw.platformWindow.keycodes[0x04B] = KeyKP4
	_glfw.platformWindow.keycodes[0x04C] = KeyKP5
	_glfw.platformWindow.keycodes[0x04D] = KeyKP6
	_glfw.platformWindow.keycodes[0x047] = KeyKP7
	_glfw.platformWindow.keycodes[0x048] = KeyKP8
	_glfw.platformWindow.keycodes[0x049] = KeyKP9
	_glfw.platformWindow.keycodes[0x04E] = KeyKPAdd
	_glfw.platformWindow.keycodes[0x053] = KeyKPDecimal
	_glfw.platformWindow.keycodes[0x135] = KeyKPDivide
	_glfw.platformWindow.keycodes[0x11C] = KeyKPEnter
	_glfw.platformWindow.keycodes[0x059] = KeyKPEqual
	_glfw.platformWindow.keycodes[0x037] = KeyKPMultiply
	_glfw.platformWindow.keycodes[0x04A] = KeyKPSubtract

	for scancode := 0; scancode < 512; scancode++ {
		if _glfw.platformWindow.keycodes[scancode] > 0 {
			_glfw.platformWindow.scancodes[_glfw.platformWindow.keycodes[scancode]] = scancode
		}
	}
}

func updateKeyNamesWin32() {
	// MapVirtualKeyW is not implemented in Xbox.
	if microsoftgdk.IsXbox() {
		return
	}

	for i := range _glfw.platformWindow.keynames {
		_glfw.platformWindow.keynames[i] = ""
	}

	var state [256]byte

	for key := KeySpace; key <= KeyLast; key++ {
		scancode := _glfw.platformWindow.scancodes[key]
		if scancode == -1 {
			continue
		}

		var vk uint32
		if key >= KeyKP0 && key <= KeyKPAdd {
			vks := []uint32{
				_VK_NUMPAD0, _VK_NUMPAD1, _VK_NUMPAD2, _VK_NUMPAD3,
				_VK_NUMPAD4, _VK_NUMPAD5, _VK_NUMPAD6, _VK_NUMPAD7,
				_VK_NUMPAD8, _VK_NUMPAD9, _VK_DECIMAL, _VK_DIVIDE,
				_VK_MULTIPLY, _VK_SUBTRACT, _VK_ADD,
			}
			vk = vks[key-KeyKP0]
		} else {
			vk = _MapVirtualKeyW(uint32(scancode), _MAPVK_VSC_TO_VK)
		}

		var chars [16]uint16
		length := _ToUnicode(vk, uint32(scancode), state[:], chars[:], int32(len(chars)), 0)
		if length == -1 {
			// This is a dead key, so we need a second simulated key press
			// to make it output its own character (usually a diacritic)
			length = _ToUnicode(vk, uint32(scancode), state[:], chars[:], int32(len(chars)), 0)
		}
		if length < 1 {
			continue
		}

		_glfw.platformWindow.keynames[key] = windows.UTF16ToString(chars[:length])
	}
}

func createHelperWindow() error {
	h, err := _CreateWindowExW(_WS_EX_OVERLAPPEDWINDOW, _GLFW_WNDCLASSNAME, "GLFW message window", _WS_CLIPSIBLINGS|_WS_CLIPCHILDREN, 0, 0, 1, 1, 0, 0, _glfw.platformWindow.instance, nil)
	if err != nil {
		return err
	}

	_glfw.platformWindow.helperWindowHandle = h

	// HACK: The command to the first ShowWindow call is ignored if the parent
	//       process passed along a STARTUPINFO, so clear that with a no-op call
	_ShowWindow(_glfw.platformWindow.helperWindowHandle, _SW_HIDE)

	// Register for HID device notifications
	if !microsoftgdk.IsXbox() {
		_GUID_DEVINTERFACE_HID := windows.GUID{
			Data1: 0x4d1e55b2,
			Data2: 0xf16f,
			Data3: 0x11cf,
			Data4: [...]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
		}

		var dbi _DEV_BROADCAST_DEVICEINTERFACE_W
		dbi.dbcc_size = uint32(unsafe.Sizeof(dbi))
		dbi.dbcc_devicetype = _DBT_DEVTYP_DEVICEINTERFACE
		dbi.dbcc_classguid = _GUID_DEVINTERFACE_HID
		notify, err := _RegisterDeviceNotificationW(windows.Handle(_glfw.platformWindow.helperWindowHandle), unsafe.Pointer(&dbi), _DEVICE_NOTIFY_WINDOW_HANDLE)
		if err != nil {
			return err
		}
		_glfw.platformWindow.deviceNotificationHandle = notify
	}

	var msg _MSG
	for _PeekMessageW(&msg, _glfw.platformWindow.helperWindowHandle, 0, 0, _PM_REMOVE) {
		_TranslateMessage(&msg)
		_DispatchMessageW(&msg)
	}

	return nil
}

func createBlankCursor() error {
	// HACK: Create a transparent cursor as using the NULL cursor breaks
	//       using SetCursorPos when connected over RDP
	cursorWidth, err := _GetSystemMetrics(_SM_CXCURSOR)
	if err != nil {
		return err
	}
	cursorHeight, err := _GetSystemMetrics(_SM_CYCURSOR)
	if err != nil {
		return err
	}
	andMask := make([]byte, cursorWidth*cursorHeight/8)
	for i := range andMask {
		andMask[i] = 0xff
	}
	xorMask := make([]byte, cursorWidth*cursorHeight/8)

	// Cursor creation might fail, but that's fine as we get NULL in that case,
	// which serves as an acceptable fallback blank cursor (other than on RDP)
	c, _ := _CreateCursor(0, 0, 0, cursorWidth, cursorHeight, andMask, xorMask)
	_glfw.platformWindow.blankCursor = c

	return nil
}

func initRemoteSession() error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	// Check if the current progress was started with Remote Desktop.
	r, err := _GetSystemMetrics(_SM_REMOTESESSION)
	if err != nil {
		return err
	}
	_glfw.platformWindow.isRemoteSession = r > 0

	// With Remote desktop, we need to create a blank cursor because of the cursor is Set to nil
	// if cannot be moved to center in capture mode. If not Remote Desktop platformWindow.blankCursor stays nil
	// and will perform has before (normal).
	if _glfw.platformWindow.isRemoteSession {
		if err := createBlankCursor(); err != nil {
			return err
		}
	}

	return nil
}

func platformInit() error {
	// Changing the foreground lock timeout was removed from the original code.
	// See https://github.com/glfw/glfw/commit/58b48a3a00d9c2a5ca10cc23069a71d8773cc7a4

	m, err := _GetModuleHandleExW(_GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS|_GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, unsafe.Pointer(&_glfw))
	if err != nil {
		return err
	}
	_glfw.platformWindow.instance = _HINSTANCE(m)

	createKeyTables()
	updateKeyNamesWin32()

	if winver.IsWindows10CreatorsUpdateOrGreater() {
		if !microsoftgdk.IsXbox() {
			// Ignore the error as SetProcessDpiAwarenessContext returns an error on Steam Deck (#2113).
			// This seems an issue in Wine and/or Proton, but there is nothing we can do.
			_ = _SetProcessDpiAwarenessContext(_DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)
		}
	} else if winver.IsWindows8Point1OrGreater() {
		if err := _SetProcessDpiAwareness(_PROCESS_PER_MONITOR_DPI_AWARE); err != nil && !errors.Is(err, handleError(windows.E_ACCESSDENIED)) {
			return err
		}
	} else if winver.IsWindowsVistaOrGreater() {
		_SetProcessDPIAware()
	}

	if err := registerWindowClassWin32(); err != nil {
		return err
	}

	if err := createHelperWindow(); err != nil {
		return err
	}
	if microsoftgdk.IsXbox() {
		// On Xbox, APIs to get monitors are not available.
		// Create a pseudo monitor instance instead.
		w, h := microsoftgdk.MonitorResolution()
		mode := &VidMode{
			Width:       w,
			Height:      h,
			RedBits:     8,
			GreenBits:   8,
			BlueBits:    8,
			RefreshRate: 0, // TODO: Is it possible to get an appropriate refresh rate?
		}
		m := &Monitor{
			name:  "Xbox Monitor",
			modes: []*VidMode{mode},
		}
		if err := inputMonitor(m, Connected, _GLFW_INSERT_LAST); err != nil {
			return err
		}
	} else {
		// Some hacks are needed to support Remote Desktop...
		if err := initRemoteSession(); err != nil {
			return err
		}
		if err := pollMonitorsWin32(); err != nil {
			return err
		}
	}
	return nil
}

func platformTerminate() error {
	if _glfw.platformWindow.blankCursor != 0 {
		if err := _DestroyCursor(_glfw.platformWindow.blankCursor); err != nil {
			return err
		}
	}

	if _glfw.platformWindow.deviceNotificationHandle != 0 {
		if err := _UnregisterDeviceNotification(_glfw.platformWindow.deviceNotificationHandle); err != nil {
			return err
		}
	}

	if _glfw.platformWindow.helperWindowHandle != 0 {
		if !microsoftgdk.IsXbox() {
			// An error 'invalid window handle' can occur without any specific reasons (#2551).
			// As there is nothing to do, just ignore this error.
			if err := _DestroyWindow(_glfw.platformWindow.helperWindowHandle); err != nil && !errors.Is(err, windows.ERROR_INVALID_WINDOW_HANDLE) {
				return err
			}
		}
	}

	if err := unregisterWindowClassWin32(); err != nil {
		return err
	}

	terminateWGL()

	return nil
}

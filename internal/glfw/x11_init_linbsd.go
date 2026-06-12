// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

// keymap maps XKB key names to GLFW keys, using the US keyboard layout.
// Because function keys aren't mapped correctly when using traditional
// KeySym translations, they are mapped here instead.
var keymap = map[string]Key{
	"TLDE": KeyGraveAccent,
	"AE01": Key1,
	"AE02": Key2,
	"AE03": Key3,
	"AE04": Key4,
	"AE05": Key5,
	"AE06": Key6,
	"AE07": Key7,
	"AE08": Key8,
	"AE09": Key9,
	"AE10": Key0,
	"AE11": KeyMinus,
	"AE12": KeyEqual,
	"AD01": KeyQ,
	"AD02": KeyW,
	"AD03": KeyE,
	"AD04": KeyR,
	"AD05": KeyT,
	"AD06": KeyY,
	"AD07": KeyU,
	"AD08": KeyI,
	"AD09": KeyO,
	"AD10": KeyP,
	"AD11": KeyLeftBracket,
	"AD12": KeyRightBracket,
	"AC01": KeyA,
	"AC02": KeyS,
	"AC03": KeyD,
	"AC04": KeyF,
	"AC05": KeyG,
	"AC06": KeyH,
	"AC07": KeyJ,
	"AC08": KeyK,
	"AC09": KeyL,
	"AC10": KeySemicolon,
	"AC11": KeyApostrophe,
	"AB01": KeyZ,
	"AB02": KeyX,
	"AB03": KeyC,
	"AB04": KeyV,
	"AB05": KeyB,
	"AB06": KeyN,
	"AB07": KeyM,
	"AB08": KeyComma,
	"AB09": KeyPeriod,
	"AB10": KeySlash,
	"BKSL": KeyBackslash,
	"LSGT": KeyWorld1,
	"SPCE": KeySpace,
	"ESC":  KeyEscape,
	"RTRN": KeyEnter,
	"TAB":  KeyTab,
	"BKSP": KeyBackspace,
	"INS":  KeyInsert,
	"DELE": KeyDelete,
	"RGHT": KeyRight,
	"LEFT": KeyLeft,
	"DOWN": KeyDown,
	"UP":   KeyUp,
	"PGUP": KeyPageUp,
	"PGDN": KeyPageDown,
	"HOME": KeyHome,
	"END":  KeyEnd,
	"CAPS": KeyCapsLock,
	"SCLK": KeyScrollLock,
	"NMLK": KeyNumLock,
	"PRSC": KeyPrintScreen,
	"PAUS": KeyPause,
	"FK01": KeyF1,
	"FK02": KeyF2,
	"FK03": KeyF3,
	"FK04": KeyF4,
	"FK05": KeyF5,
	"FK06": KeyF6,
	"FK07": KeyF7,
	"FK08": KeyF8,
	"FK09": KeyF9,
	"FK10": KeyF10,
	"FK11": KeyF11,
	"FK12": KeyF12,
	"FK13": KeyF13,
	"FK14": KeyF14,
	"FK15": KeyF15,
	"FK16": KeyF16,
	"FK17": KeyF17,
	"FK18": KeyF18,
	"FK19": KeyF19,
	"FK20": KeyF20,
	"FK21": KeyF21,
	"FK22": KeyF22,
	"FK23": KeyF23,
	"FK24": KeyF24,
	"KP0":  KeyKP0,
	"KP1":  KeyKP1,
	"KP2":  KeyKP2,
	"KP3":  KeyKP3,
	"KP4":  KeyKP4,
	"KP5":  KeyKP5,
	"KP6":  KeyKP6,
	"KP7":  KeyKP7,
	"KP8":  KeyKP8,
	"KP9":  KeyKP9,
	"KPDL": KeyKPDecimal,
	"KPDV": KeyKPDivide,
	"KPMU": KeyKPMultiply,
	"KPSU": KeyKPSubtract,
	"KPAD": KeyKPAdd,
	"KPEN": KeyKPEnter,
	"KPEQ": KeyKPEqual,
	"LFSH": KeyLeftShift,
	"LCTL": KeyLeftControl,
	"LALT": KeyLeftAlt,
	"LWIN": KeyLeftSuper,
	"RTSH": KeyRightShift,
	"RCTL": KeyRightControl,
	"RALT": KeyRightAlt,
	"LVL3": KeyRightAlt,
	"MDSW": KeyRightAlt,
	"RWIN": KeyRightSuper,
	"MENU": KeyMenu,
}

// keyNameString converts an XKB key name, which is not NUL-terminated when
// all four bytes are used, into a string.
func keyNameString(name [4]byte) string {
	for i, c := range name {
		if c == 0 {
			return string(name[:i])
		}
	}
	return string(name[:])
}

// translateKeySyms translates an X11 KeySym row to a GLFW key.
// NOTE: This is only used as a fallback, in case the XKB method fails
//
//	It is layout-dependent and will fail partially on most non-US layouts
func translateKeySyms(keysyms []KeySym) Key {
	if len(keysyms) > 1 {
		switch keysyms[1] {
		case XK_KP_0:
			return KeyKP0
		case XK_KP_1:
			return KeyKP1
		case XK_KP_2:
			return KeyKP2
		case XK_KP_3:
			return KeyKP3
		case XK_KP_4:
			return KeyKP4
		case XK_KP_5:
			return KeyKP5
		case XK_KP_6:
			return KeyKP6
		case XK_KP_7:
			return KeyKP7
		case XK_KP_8:
			return KeyKP8
		case XK_KP_9:
			return KeyKP9
		case XK_KP_Separator, XK_KP_Decimal:
			return KeyKPDecimal
		case XK_KP_Equal:
			return KeyKPEqual
		case XK_KP_Enter:
			return KeyKPEnter
		}
	}

	switch keysyms[0] {
	case XK_Escape:
		return KeyEscape
	case XK_Tab:
		return KeyTab
	case XK_Shift_L:
		return KeyLeftShift
	case XK_Shift_R:
		return KeyRightShift
	case XK_Control_L:
		return KeyLeftControl
	case XK_Control_R:
		return KeyRightControl
	case XK_Meta_L, XK_Alt_L:
		return KeyLeftAlt
	case XK_Mode_switch, // Mapped to Alt_R on many keyboards
		XK_ISO_Level3_Shift, // AltGr on at least some machines
		XK_Meta_R,
		XK_Alt_R:
		return KeyRightAlt
	case XK_Super_L:
		return KeyLeftSuper
	case XK_Super_R:
		return KeyRightSuper
	case XK_Menu:
		return KeyMenu
	case XK_Num_Lock:
		return KeyNumLock
	case XK_Caps_Lock:
		return KeyCapsLock
	case XK_Print:
		return KeyPrintScreen
	case XK_Scroll_Lock:
		return KeyScrollLock
	case XK_Pause:
		return KeyPause
	case XK_Delete:
		return KeyDelete
	case XK_BackSpace:
		return KeyBackspace
	case XK_Return:
		return KeyEnter
	case XK_Home:
		return KeyHome
	case XK_End:
		return KeyEnd
	case XK_Page_Up:
		return KeyPageUp
	case XK_Page_Down:
		return KeyPageDown
	case XK_Insert:
		return KeyInsert
	case XK_Left:
		return KeyLeft
	case XK_Right:
		return KeyRight
	case XK_Down:
		return KeyDown
	case XK_Up:
		return KeyUp
	case XK_F1:
		return KeyF1
	case XK_F2:
		return KeyF2
	case XK_F3:
		return KeyF3
	case XK_F4:
		return KeyF4
	case XK_F5:
		return KeyF5
	case XK_F6:
		return KeyF6
	case XK_F7:
		return KeyF7
	case XK_F8:
		return KeyF8
	case XK_F9:
		return KeyF9
	case XK_F10:
		return KeyF10
	case XK_F11:
		return KeyF11
	case XK_F12:
		return KeyF12
	case XK_F13:
		return KeyF13
	case XK_F14:
		return KeyF14
	case XK_F15:
		return KeyF15
	case XK_F16:
		return KeyF16
	case XK_F17:
		return KeyF17
	case XK_F18:
		return KeyF18
	case XK_F19:
		return KeyF19
	case XK_F20:
		return KeyF20
	case XK_F21:
		return KeyF21
	case XK_F22:
		return KeyF22
	case XK_F23:
		return KeyF23
	case XK_F24:
		return KeyF24

	// Numeric keypad
	case XK_KP_Divide:
		return KeyKPDivide
	case XK_KP_Multiply:
		return KeyKPMultiply
	case XK_KP_Subtract:
		return KeyKPSubtract
	case XK_KP_Add:
		return KeyKPAdd

	// These should have been detected in secondary keysym test above!
	case XK_KP_Insert:
		return KeyKP0
	case XK_KP_End:
		return KeyKP1
	case XK_KP_Down:
		return KeyKP2
	case XK_KP_Page_Down:
		return KeyKP3
	case XK_KP_Left:
		return KeyKP4
	case XK_KP_Right:
		return KeyKP6
	case XK_KP_Home:
		return KeyKP7
	case XK_KP_Up:
		return KeyKP8
	case XK_KP_Page_Up:
		return KeyKP9
	case XK_KP_Delete:
		return KeyKPDecimal
	case XK_KP_Equal:
		return KeyKPEqual
	case XK_KP_Enter:
		return KeyKPEnter

	// Last resort: Check for printable keys (should not happen if the XKB
	// extension is available). This will give a layout dependent mapping
	// (which is wrong, and we may miss some keys, especially on non-US
	// keyboards), but it's better than nothing...
	case XK_a:
		return KeyA
	case XK_b:
		return KeyB
	case XK_c:
		return KeyC
	case XK_d:
		return KeyD
	case XK_e:
		return KeyE
	case XK_f:
		return KeyF
	case XK_g:
		return KeyG
	case XK_h:
		return KeyH
	case XK_i:
		return KeyI
	case XK_j:
		return KeyJ
	case XK_k:
		return KeyK
	case XK_l:
		return KeyL
	case XK_m:
		return KeyM
	case XK_n:
		return KeyN
	case XK_o:
		return KeyO
	case XK_p:
		return KeyP
	case XK_q:
		return KeyQ
	case XK_r:
		return KeyR
	case XK_s:
		return KeyS
	case XK_t:
		return KeyT
	case XK_u:
		return KeyU
	case XK_v:
		return KeyV
	case XK_w:
		return KeyW
	case XK_x:
		return KeyX
	case XK_y:
		return KeyY
	case XK_z:
		return KeyZ
	case XK_1:
		return Key1
	case XK_2:
		return Key2
	case XK_3:
		return Key3
	case XK_4:
		return Key4
	case XK_5:
		return Key5
	case XK_6:
		return Key6
	case XK_7:
		return Key7
	case XK_8:
		return Key8
	case XK_9:
		return Key9
	case XK_0:
		return Key0
	case XK_space:
		return KeySpace
	case XK_minus:
		return KeyMinus
	case XK_equal:
		return KeyEqual
	case XK_bracketleft:
		return KeyLeftBracket
	case XK_bracketright:
		return KeyRightBracket
	case XK_backslash:
		return KeyBackslash
	case XK_semicolon:
		return KeySemicolon
	case XK_apostrophe:
		return KeyApostrophe
	case XK_grave:
		return KeyGraveAccent
	case XK_comma:
		return KeyComma
	case XK_period:
		return KeyPeriod
	case XK_slash:
		return KeySlash
	case XK_less:
		return KeyWorld1 // At least in some layouts...
	}

	// No matching translation was found
	return KeyUnknown
}

// createKeyTables creates the key code translation tables.
func createKeyTables() {
	for i := range _glfw.platformWindow.keycodes {
		_glfw.platformWindow.keycodes[i] = KeyUnknown
	}
	for i := range _glfw.platformWindow.scancodes {
		_glfw.platformWindow.scancodes[i] = -1
	}

	var scancodeMin, scancodeMax int32
	if _glfw.platformWindow.xkb.available {
		// Use XKB to determine physical key locations independently of the
		// current keyboard layout

		descPtr := xkbGetMap(_glfw.platformWindow.display, 0, XkbUseCoreKbd)
		xkbGetNames(_glfw.platformWindow.display, XkbKeyNamesMask|XkbKeyAliasesMask, descPtr)

		desc := (*XkbDescRec)(unsafe.Pointer(descPtr))
		scancodeMin = int32(desc.MinKeyCode)
		scancodeMax = int32(desc.MaxKeyCode)

		names := (*XkbNamesRec)(unsafe.Pointer(desc.Names))
		keyNames := unsafe.Slice((*XkbKeyNameRec)(unsafe.Pointer(names.Keys)), int(scancodeMax)+1)
		keyAliases := unsafe.Slice((*XkbKeyAliasRec)(unsafe.Pointer(names.KeyAliases)), int(names.NumKeyAliases))

		// Find the X11 key code -> GLFW key code mapping
		for scancode := scancodeMin; scancode <= scancodeMax; scancode++ {
			key := KeyUnknown

			// Map the key name to a GLFW key code. Note: We use the US
			// keyboard layout. Because function keys aren't mapped correctly
			// when using traditional KeySym translations, they are mapped
			// here instead.
			if k, ok := keymap[keyNameString(keyNames[scancode].Name)]; ok {
				key = k
			}

			// Fall back to key aliases in case the key name did not match
			for i := range keyAliases {
				if key != KeyUnknown {
					break
				}
				if keyAliases[i].Real != keyNames[scancode].Name {
					continue
				}
				if k, ok := keymap[keyNameString(keyAliases[i].Alias)]; ok {
					key = k
					break
				}
			}

			_glfw.platformWindow.keycodes[scancode] = key
		}

		xkbFreeNames(descPtr, XkbKeyNamesMask, true)
		xkbFreeKeyboard(descPtr, 0, true)
	} else {
		xDisplayKeycodes(_glfw.platformWindow.display, &scancodeMin, &scancodeMax)
	}

	var width int32
	keysymsPtr := xGetKeyboardMapping(_glfw.platformWindow.display,
		KeyCode(scancodeMin),
		scancodeMax-scancodeMin+1,
		&width)
	keysyms := unsafe.Slice((*KeySym)(unsafe.Pointer(keysymsPtr)), int(scancodeMax-scancodeMin+1)*int(width))

	for scancode := scancodeMin; scancode <= scancodeMax; scancode++ {
		// Translate the un-translated key codes using traditional X11 KeySym
		// lookups
		if _glfw.platformWindow.keycodes[scancode] < 0 {
			base := int(scancode-scancodeMin) * int(width)
			_glfw.platformWindow.keycodes[scancode] = translateKeySyms(keysyms[base : base+int(width)])
		}

		// Store the reverse translation for faster key name lookup
		if key := _glfw.platformWindow.keycodes[scancode]; key > 0 {
			_glfw.platformWindow.scancodes[key] = int(scancode)
		}
	}

	xFree(keysymsPtr)
}

// hasUsableInputMethodStyle reports whether the IM has a usable style.
func hasUsableInputMethodStyle() bool {
	found := false
	var stylesPtr uintptr

	if xGetIMValues(_glfw.platformWindow.im, "queryInputStyle", &stylesPtr, 0) != 0 {
		return false
	}

	styles := (*XIMStyles)(unsafe.Pointer(stylesPtr))
	supportedStyles := unsafe.Slice((*XIMStyle)(unsafe.Pointer(styles.SupportedStyles)), int(styles.CountStyles))
	for _, style := range supportedStyles {
		if style == XIMPreeditNothing|XIMStatusNothing {
			found = true
			break
		}
	}

	xFree(stylesPtr)
	return found
}

// getAtomIfSupported returns the atom with the specified name if it is
// supported, and None otherwise.
func getAtomIfSupported(supportedAtoms []Atom, atomName string) Atom {
	atom := xInternAtom(_glfw.platformWindow.display, atomName, false)
	for _, supported := range supportedAtoms {
		if supported == atom {
			return atom
		}
	}
	return None
}

// detectEWMH checks whether the running window manager is EWMH-compliant.
func detectEWMH() {
	// First we read the _NET_SUPPORTING_WM_CHECK property on the root window

	var windowFromRootPtr uintptr
	if getWindowPropertyX11(_glfw.platformWindow.root,
		_glfw.platformWindow.NET_SUPPORTING_WM_CHECK,
		XA_WINDOW,
		&windowFromRootPtr) == 0 {
		return
	}

	grabErrorHandlerX11()

	// If it exists, it should be the XID of a top-level window
	// Then we look for the same property on that window

	var windowFromChildPtr uintptr
	if getWindowPropertyX11(*(*XID)(unsafe.Pointer(windowFromRootPtr)),
		_glfw.platformWindow.NET_SUPPORTING_WM_CHECK,
		XA_WINDOW,
		&windowFromChildPtr) == 0 {
		xFree(windowFromRootPtr)
		return
	}

	releaseErrorHandlerX11()

	// If the property exists, it should contain the XID of the window

	if *(*XID)(unsafe.Pointer(windowFromRootPtr)) != *(*XID)(unsafe.Pointer(windowFromChildPtr)) {
		xFree(windowFromRootPtr)
		xFree(windowFromChildPtr)
		return
	}

	xFree(windowFromRootPtr)
	xFree(windowFromChildPtr)

	// We are now fairly sure that an EWMH-compliant WM is currently running
	// We can now start querying the WM about what features it supports by
	// looking in the _NET_SUPPORTED property on the root window
	// It should contain a list of supported EWMH protocol and state atoms

	var supportedAtomsPtr uintptr
	atomCount := getWindowPropertyX11(_glfw.platformWindow.root,
		_glfw.platformWindow.NET_SUPPORTED,
		XA_ATOM,
		&supportedAtomsPtr)

	var supportedAtoms []Atom
	if supportedAtomsPtr != 0 {
		supportedAtoms = unsafe.Slice((*Atom)(unsafe.Pointer(supportedAtomsPtr)), int(atomCount))
	}

	// See which of the atoms we support that are supported by the WM

	_glfw.platformWindow.NET_WM_STATE =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE")
	_glfw.platformWindow.NET_WM_STATE_ABOVE =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE_ABOVE")
	_glfw.platformWindow.NET_WM_STATE_FULLSCREEN =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE_FULLSCREEN")
	_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_VERT =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE_MAXIMIZED_VERT")
	_glfw.platformWindow.NET_WM_STATE_MAXIMIZED_HORZ =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE_MAXIMIZED_HORZ")
	_glfw.platformWindow.NET_WM_STATE_DEMANDS_ATTENTION =
		getAtomIfSupported(supportedAtoms, "_NET_WM_STATE_DEMANDS_ATTENTION")
	_glfw.platformWindow.NET_WM_FULLSCREEN_MONITORS =
		getAtomIfSupported(supportedAtoms, "_NET_WM_FULLSCREEN_MONITORS")
	_glfw.platformWindow.NET_WM_WINDOW_TYPE =
		getAtomIfSupported(supportedAtoms, "_NET_WM_WINDOW_TYPE")
	_glfw.platformWindow.NET_WM_WINDOW_TYPE_NORMAL =
		getAtomIfSupported(supportedAtoms, "_NET_WM_WINDOW_TYPE_NORMAL")
	_glfw.platformWindow.NET_WORKAREA =
		getAtomIfSupported(supportedAtoms, "_NET_WORKAREA")
	_glfw.platformWindow.NET_CURRENT_DESKTOP =
		getAtomIfSupported(supportedAtoms, "_NET_CURRENT_DESKTOP")
	_glfw.platformWindow.NET_ACTIVE_WINDOW =
		getAtomIfSupported(supportedAtoms, "_NET_ACTIVE_WINDOW")
	_glfw.platformWindow.NET_FRAME_EXTENTS =
		getAtomIfSupported(supportedAtoms, "_NET_FRAME_EXTENTS")
	_glfw.platformWindow.NET_REQUEST_FRAME_EXTENTS =
		getAtomIfSupported(supportedAtoms, "_NET_REQUEST_FRAME_EXTENTS")

	if supportedAtomsPtr != 0 {
		xFree(supportedAtomsPtr)
	}
}

// initExtensions looks for and initializes supported X11 extensions.
func initExtensions() error {
	display := _glfw.platformWindow.display

	if handle, err := openX11Library("libXi.so.6", "libXi.so"); err == nil {
		xi := &_glfw.platformWindow.xi
		xi.handle = handle
		purego.RegisterLibFunc(&xi.QueryVersion, handle, "XIQueryVersion")
		purego.RegisterLibFunc(&xi.SelectEvents, handle, "XISelectEvents")

		if xQueryExtension(display, "XInputExtension", &xi.majorOpcode, &xi.eventBase, &xi.errorBase) {
			xi.major = 2
			xi.minor = 0
			if xi.QueryVersion(display, &xi.major, &xi.minor) == Success {
				xi.available = true
			}
		}
	}

	if handle, err := openX11Library("libXrandr.so.2", "libXrandr.so"); err == nil {
		randr := &_glfw.platformWindow.randr
		randr.handle = handle
		purego.RegisterLibFunc(&randr.FreeCrtcInfo, handle, "XRRFreeCrtcInfo")
		purego.RegisterLibFunc(&randr.FreeOutputInfo, handle, "XRRFreeOutputInfo")
		purego.RegisterLibFunc(&randr.FreeScreenResources, handle, "XRRFreeScreenResources")
		purego.RegisterLibFunc(&randr.GetCrtcInfo, handle, "XRRGetCrtcInfo")
		purego.RegisterLibFunc(&randr.GetOutputInfo, handle, "XRRGetOutputInfo")
		purego.RegisterLibFunc(&randr.GetOutputPrimary, handle, "XRRGetOutputPrimary")
		purego.RegisterLibFunc(&randr.GetScreenResourcesCurrent, handle, "XRRGetScreenResourcesCurrent")
		purego.RegisterLibFunc(&randr.QueryExtension, handle, "XRRQueryExtension")
		purego.RegisterLibFunc(&randr.QueryVersion, handle, "XRRQueryVersion")
		purego.RegisterLibFunc(&randr.SelectInput, handle, "XRRSelectInput")
		purego.RegisterLibFunc(&randr.SetCrtcConfig, handle, "XRRSetCrtcConfig")
		purego.RegisterLibFunc(&randr.UpdateConfiguration, handle, "XRRUpdateConfiguration")

		if randr.QueryExtension(display, &randr.eventBase, &randr.errorBase) {
			if randr.QueryVersion(display, &randr.major, &randr.minor) != 0 {
				// The GLFW RandR path requires at least version 1.3
				if randr.major > 1 || randr.minor >= 3 {
					randr.available = true
				}
			}
		}
	}

	if _glfw.platformWindow.randr.available {
		randr := &_glfw.platformWindow.randr
		sr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)

		if (*XRRScreenResources)(unsafe.Pointer(sr)).Ncrtc == 0 {
			// A system without CRTCs is likely a system with broken RandR
			// Disable the RandR monitor path and fall back to core functions
			randr.monitorBroken = true
		}

		randr.FreeScreenResources(sr)
	}

	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		_glfw.platformWindow.randr.SelectInput(display, _glfw.platformWindow.root, RROutputChangeNotifyMask)
	}

	if handle, err := openX11Library("libXcursor.so.1", "libXcursor.so"); err == nil {
		xcursor := &_glfw.platformWindow.xcursor
		xcursor.handle = handle
		purego.RegisterLibFunc(&xcursor.ImageCreate, handle, "XcursorImageCreate")
		purego.RegisterLibFunc(&xcursor.ImageDestroy, handle, "XcursorImageDestroy")
		purego.RegisterLibFunc(&xcursor.ImageLoadCursor, handle, "XcursorImageLoadCursor")
		purego.RegisterLibFunc(&xcursor.GetTheme, handle, "XcursorGetTheme")
		purego.RegisterLibFunc(&xcursor.GetDefaultSize, handle, "XcursorGetDefaultSize")
		purego.RegisterLibFunc(&xcursor.LibraryLoadImage, handle, "XcursorLibraryLoadImage")
	}

	if handle, err := openX11Library("libXinerama.so.1", "libXinerama.so"); err == nil {
		xinerama := &_glfw.platformWindow.xinerama
		xinerama.handle = handle
		purego.RegisterLibFunc(&xinerama.IsActive, handle, "XineramaIsActive")
		purego.RegisterLibFunc(&xinerama.QueryExtension, handle, "XineramaQueryExtension")
		purego.RegisterLibFunc(&xinerama.QueryScreens, handle, "XineramaQueryScreens")

		if xinerama.QueryExtension(display, &xinerama.major, &xinerama.minor) {
			if xinerama.IsActive(display) {
				xinerama.available = true
			}
		}
	}

	xkb := &_glfw.platformWindow.xkb
	xkb.major = 1
	xkb.minor = 0
	xkb.available = xkbQueryExtension(display, &xkb.majorOpcode, &xkb.eventBase, &xkb.errorBase, &xkb.major, &xkb.minor)

	if xkb.available {
		var supported int32
		if xkbSetDetectableAutoRepeat(display, true, &supported) {
			if supported != 0 {
				xkb.detectable = true
			}
		}

		var state XkbStateRec
		if xkbGetState(display, XkbUseCoreKbd, &state) == Success {
			xkb.group = uint32(state.Group)
		}

		xkbSelectEventDetails(display, XkbUseCoreKbd, XkbStateNotify, XkbGroupStateMask, XkbGroupStateMask)
	}

	if handle, err := openX11Library("libXrender.so.1", "libXrender.so"); err == nil {
		xrender := &_glfw.platformWindow.xrender
		xrender.handle = handle
		purego.RegisterLibFunc(&xrender.QueryExtension, handle, "XRenderQueryExtension")
		purego.RegisterLibFunc(&xrender.QueryVersion, handle, "XRenderQueryVersion")
		purego.RegisterLibFunc(&xrender.FindVisualFormat, handle, "XRenderFindVisualFormat")

		if xrender.QueryExtension(display, &xrender.errorBase, &xrender.eventBase) {
			if xrender.QueryVersion(display, &xrender.major, &xrender.minor) != 0 {
				xrender.available = true
			}
		}
	}

	if handle, err := openX11Library("libXext.so.6", "libXext.so"); err == nil {
		xshape := &_glfw.platformWindow.xshape
		xshape.handle = handle
		purego.RegisterLibFunc(&xshape.QueryExtension, handle, "XShapeQueryExtension")
		purego.RegisterLibFunc(&xshape.CombineRegion, handle, "XShapeCombineRegion")
		purego.RegisterLibFunc(&xshape.QueryVersion, handle, "XShapeQueryVersion")
		purego.RegisterLibFunc(&xshape.CombineMask, handle, "XShapeCombineMask")

		if xshape.QueryExtension(display, &xshape.errorBase, &xshape.eventBase) {
			if xshape.QueryVersion(display, &xshape.major, &xshape.minor) != 0 {
				xshape.available = true
			}
		}
	}

	// Update the key code LUT
	// FIXME: We should listen to XkbMapNotify events to track changes to
	// the keyboard mapping.
	createKeyTables()

	// String format atoms
	_glfw.platformWindow.NULL_ = xInternAtom(display, "NULL", false)
	_glfw.platformWindow.UTF8_STRING = xInternAtom(display, "UTF8_STRING", false)
	_glfw.platformWindow.ATOM_PAIR = xInternAtom(display, "ATOM_PAIR", false)

	// Custom selection property atom
	_glfw.platformWindow.GLFW_SELECTION = xInternAtom(display, "GLFW_SELECTION", false)

	// ICCCM standard clipboard atoms
	_glfw.platformWindow.TARGETS = xInternAtom(display, "TARGETS", false)
	_glfw.platformWindow.MULTIPLE = xInternAtom(display, "MULTIPLE", false)
	_glfw.platformWindow.PRIMARY = xInternAtom(display, "PRIMARY", false)
	_glfw.platformWindow.INCR = xInternAtom(display, "INCR", false)
	_glfw.platformWindow.CLIPBOARD = xInternAtom(display, "CLIPBOARD", false)

	// Clipboard manager atoms
	_glfw.platformWindow.CLIPBOARD_MANAGER = xInternAtom(display, "CLIPBOARD_MANAGER", false)
	_glfw.platformWindow.SAVE_TARGETS = xInternAtom(display, "SAVE_TARGETS", false)

	// Xdnd (drag and drop) atoms
	_glfw.platformWindow.XdndAware = xInternAtom(display, "XdndAware", false)
	_glfw.platformWindow.XdndEnter = xInternAtom(display, "XdndEnter", false)
	_glfw.platformWindow.XdndPosition = xInternAtom(display, "XdndPosition", false)
	_glfw.platformWindow.XdndStatus = xInternAtom(display, "XdndStatus", false)
	_glfw.platformWindow.XdndActionCopy = xInternAtom(display, "XdndActionCopy", false)
	_glfw.platformWindow.XdndDrop = xInternAtom(display, "XdndDrop", false)
	_glfw.platformWindow.XdndFinished = xInternAtom(display, "XdndFinished", false)
	_glfw.platformWindow.XdndSelection = xInternAtom(display, "XdndSelection", false)
	_glfw.platformWindow.XdndTypeList = xInternAtom(display, "XdndTypeList", false)
	_glfw.platformWindow.text_uri_list = xInternAtom(display, "text/uri-list", false)

	// ICCCM, EWMH and Motif window property atoms
	// These can be set safely even without WM support
	// The EWMH atoms that require WM support are handled in detectEWMH
	_glfw.platformWindow.WM_PROTOCOLS = xInternAtom(display, "WM_PROTOCOLS", false)
	_glfw.platformWindow.WM_STATE = xInternAtom(display, "WM_STATE", false)
	_glfw.platformWindow.WM_DELETE_WINDOW = xInternAtom(display, "WM_DELETE_WINDOW", false)
	_glfw.platformWindow.NET_SUPPORTED = xInternAtom(display, "_NET_SUPPORTED", false)
	_glfw.platformWindow.NET_SUPPORTING_WM_CHECK = xInternAtom(display, "_NET_SUPPORTING_WM_CHECK", false)
	_glfw.platformWindow.NET_WM_ICON = xInternAtom(display, "_NET_WM_ICON", false)
	_glfw.platformWindow.NET_WM_PING = xInternAtom(display, "_NET_WM_PING", false)
	_glfw.platformWindow.NET_WM_PID = xInternAtom(display, "_NET_WM_PID", false)
	_glfw.platformWindow.NET_WM_NAME = xInternAtom(display, "_NET_WM_NAME", false)
	_glfw.platformWindow.NET_WM_ICON_NAME = xInternAtom(display, "_NET_WM_ICON_NAME", false)
	_glfw.platformWindow.NET_WM_BYPASS_COMPOSITOR = xInternAtom(display, "_NET_WM_BYPASS_COMPOSITOR", false)
	_glfw.platformWindow.NET_WM_WINDOW_OPACITY = xInternAtom(display, "_NET_WM_WINDOW_OPACITY", false)
	_glfw.platformWindow.MOTIF_WM_HINTS = xInternAtom(display, "_MOTIF_WM_HINTS", false)

	// The compositing manager selection name contains the screen number
	_glfw.platformWindow.NET_WM_CM_Sx = xInternAtom(display, fmt.Sprintf("_NET_WM_CM_S%d", _glfw.platformWindow.screen), false)

	// Detect whether an EWMH-conformant window manager is running
	detectEWMH()

	return nil
}

// getSystemContentScale retrieves the system content scale via folklore
// heuristics.
func getSystemContentScale() (xscale, yscale float32) {
	// Start by assuming the default X11 DPI
	// NOTE: Some desktop environments (KDE) may remove the Xft.dpi field when it
	//       would be set to 96, so assume that is the case if we cannot find it
	xdpi, ydpi := float32(96), float32(96)

	// NOTE: Basing the scale on Xft.dpi where available should provide the most
	//       consistent user experience (matches Qt, Gtk, etc), although not
	//       always the most accurate one
	rms := xResourceManagerString(_glfw.platformWindow.display)
	if rms != 0 {
		db := xrmGetStringDatabase(rms)
		if db != 0 {
			var value XrmValue
			var typ uintptr
			if xrmGetResource(db, "Xft.dpi", "Xft.Dpi", &typ, &value) {
				if typ != 0 && goString(typ) == "String" {
					if dpi, err := strconv.ParseFloat(goString(value.Addr), 32); err == nil {
						xdpi = float32(dpi)
						ydpi = float32(dpi)
					}
				}
			}
			xrmDestroyDatabase(db)
		}
	}

	return xdpi / 96, ydpi / 96
}

// createHiddenCursor creates a blank cursor for hidden and disabled cursor
// modes.
func createHiddenCursor() XID {
	pixels := make([]byte, 16*16*4)
	return createCursorX11(&Image{Width: 16, Height: 16, Pixels: pixels}, 0, 0)
}

// createHelperWindow creates a helper window for IPC.
func createHelperWindow() XID {
	var wa XSetWindowAttributes
	wa.EventMask = PropertyChangeMask

	return xCreateWindow(_glfw.platformWindow.display, _glfw.platformWindow.root,
		0, 0, 1, 1, 0, 0,
		InputOnly,
		xDefaultVisual(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen)),
		CWEventMask, &wa)
}

// createEmptyEventPipe creates the pipe for empty events.
func createEmptyEventPipe() error {
	if err := unix.Pipe2(_glfw.platformWindow.emptyEventPipe[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		return fmt.Errorf("glfw: x11: failed to create empty event pipe: %v: %w", err, PlatformError)
	}
	return nil
}

// errorHandlerCallback is the purego callback pointer for errorHandler.
var errorHandlerCallback uintptr

// errorHandler is the X error handler.
func errorHandler(display uintptr, event uintptr) uintptr {
	if _glfw.platformWindow.display != display {
		return 0
	}

	_glfw.platformWindow.errorCode = int((*XErrorEvent)(unsafe.Pointer(event)).ErrorCode)
	return 0
}

// grabErrorHandlerX11 sets the X error handler callback.
func grabErrorHandlerX11() {
	if errorHandlerCallback == 0 {
		errorHandlerCallback = purego.NewCallback(errorHandler)
	}
	_glfw.platformWindow.errorCode = Success
	_glfw.platformWindow.errorHandler = xSetErrorHandler(errorHandlerCallback)
}

// releaseErrorHandlerX11 clears the X error handler callback.
func releaseErrorHandlerX11() {
	// Synchronize to make sure all commands are processed
	xSync(_glfw.platformWindow.display, false)
	xSetErrorHandler(_glfw.platformWindow.errorHandler)
	_glfw.platformWindow.errorHandler = 0
}

// inputErrorX11 returns the specified error, appending information about the
// last X error.
func inputErrorX11(code ErrorCode, message string) error {
	buffer := make([]byte, 1024)
	xGetErrorText(_glfw.platformWindow.display, int32(_glfw.platformWindow.errorCode), buffer, int32(len(buffer)))
	text := buffer[:]
	for i, c := range buffer {
		if c == 0 {
			text = buffer[:i]
			break
		}
	}
	return fmt.Errorf("glfw: %s: %s: %w", message, text, code)
}

// createCursorX11 creates a native cursor object from the specified image and
// hotspot.
func createCursorX11(image *Image, xhot, yhot int) XID {
	if _glfw.platformWindow.xcursor.handle == 0 {
		return None
	}

	nativePtr := _glfw.platformWindow.xcursor.ImageCreate(int32(image.Width), int32(image.Height))
	if nativePtr == 0 {
		return None
	}
	native := (*XcursorImage)(unsafe.Pointer(nativePtr))

	native.Xhot = uint32(xhot)
	native.Yhot = uint32(yhot)

	source := image.Pixels
	target := unsafe.Slice((*uint32)(unsafe.Pointer(native.Pixels)), image.Width*image.Height)

	for i := range target {
		alpha := uint32(source[i*4+3])

		target[i] = alpha<<24 |
			(uint32(source[i*4])*alpha)/255<<16 |
			(uint32(source[i*4+1])*alpha)/255<<8 |
			(uint32(source[i*4+2])*alpha)/255<<0
	}

	cursor := _glfw.platformWindow.xcursor.ImageLoadCursor(_glfw.platformWindow.display, nativePtr)
	_glfw.platformWindow.xcursor.ImageDestroy(nativePtr)

	return cursor
}

func platformInit() error {
	if err := initLibX11(); err != nil {
		return err
	}

	// HACK: If the application has left the locale as "C" then both wide
	//       character text input and explicit UTF-8 input via XIM will break
	//       This sets the CTYPE part of the current locale from the environment
	//       in the hope that it is set to something more sane than "C"
	if goString(setlocaleQuery(lcCType, 0)) == "C" {
		setlocale(lcCType, "")
	}

	xInitThreads()
	xrmInitialize()

	_glfw.platformWindow.display = xOpenDisplay(0)
	if _glfw.platformWindow.display == 0 {
		if display := os.Getenv("DISPLAY"); display != "" {
			return fmt.Errorf("glfw: x11: failed to open display %s: %w", display, PlatformError)
		}
		return fmt.Errorf("glfw: x11: the DISPLAY environment variable is missing: %w", PlatformError)
	}

	_glfw.platformWindow.screen = int(xDefaultScreen(_glfw.platformWindow.display))
	_glfw.platformWindow.root = xRootWindow(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen))
	_glfw.platformWindow.windowsByXID = map[XID]*Window{}

	_glfw.platformWindow.contentScaleX, _glfw.platformWindow.contentScaleY = getSystemContentScale()

	if err := createEmptyEventPipe(); err != nil {
		return err
	}

	if err := initExtensions(); err != nil {
		return err
	}

	_glfw.platformWindow.helperWindowHandle = createHelperWindow()
	_glfw.platformWindow.hiddenCursorHandle = createHiddenCursor()

	if xSupportsLocale() {
		xSetLocaleModifiers("")

		_glfw.platformWindow.im = xOpenIM(_glfw.platformWindow.display, 0, 0, 0)
		if _glfw.platformWindow.im != 0 {
			if !hasUsableInputMethodStyle() {
				xCloseIM(_glfw.platformWindow.im)
				_glfw.platformWindow.im = 0
			}
		}
	}

	if err := pollMonitorsX11(); err != nil {
		return err
	}
	return nil
}

func platformTerminate() error {
	if _glfw.platformWindow.helperWindowHandle != 0 {
		if xGetSelectionOwner(_glfw.platformWindow.display, _glfw.platformWindow.CLIPBOARD) == _glfw.platformWindow.helperWindowHandle {
			pushSelectionToManagerX11()
		}

		xDestroyWindow(_glfw.platformWindow.display, _glfw.platformWindow.helperWindowHandle)
		_glfw.platformWindow.helperWindowHandle = None
	}

	if _glfw.platformWindow.hiddenCursorHandle != 0 {
		xFreeCursor(_glfw.platformWindow.display, _glfw.platformWindow.hiddenCursorHandle)
		_glfw.platformWindow.hiddenCursorHandle = 0
	}

	_glfw.platformWindow.primarySelectionString = ""
	_glfw.platformWindow.clipboardString = ""

	if _glfw.platformWindow.im != 0 {
		xCloseIM(_glfw.platformWindow.im)
		_glfw.platformWindow.im = 0
	}

	if _glfw.platformWindow.display != 0 {
		xCloseDisplay(_glfw.platformWindow.display)
		_glfw.platformWindow.display = 0
	}

	if _glfw.platformWindow.xcursor.handle != 0 {
		_ = purego.Dlclose(_glfw.platformWindow.xcursor.handle)
		_glfw.platformWindow.xcursor.handle = 0
	}

	if _glfw.platformWindow.randr.handle != 0 {
		_ = purego.Dlclose(_glfw.platformWindow.randr.handle)
		_glfw.platformWindow.randr.handle = 0
	}

	if _glfw.platformWindow.xinerama.handle != 0 {
		_ = purego.Dlclose(_glfw.platformWindow.xinerama.handle)
		_glfw.platformWindow.xinerama.handle = 0
	}

	if _glfw.platformWindow.xrender.handle != 0 {
		_ = purego.Dlclose(_glfw.platformWindow.xrender.handle)
		_glfw.platformWindow.xrender.handle = 0
	}

	if _glfw.platformWindow.xi.handle != 0 {
		_ = purego.Dlclose(_glfw.platformWindow.xi.handle)
		_glfw.platformWindow.xi.handle = 0
	}

	// NOTE: These need to be unloaded after XCloseDisplay, as they register
	//       cleanup callbacks that get called by that function
	terminateEGL()
	terminateGLX()

	if _glfw.platformWindow.emptyEventPipe[0] != 0 || _glfw.platformWindow.emptyEventPipe[1] != 0 {
		_ = unix.Close(_glfw.platformWindow.emptyEventPipe[0])
		_ = unix.Close(_glfw.platformWindow.emptyEventPipe[1])
	}
	return nil
}

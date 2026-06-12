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

//go:build freebsd || linux || netbsd

package glfw

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// This file binds the libX11 core functions via purego. The X extension
// libraries (Xrandr, Xcursor, Xinerama, Xi, Xrender, Xext) are loaded
// separately into the X11 library state, mirroring the C implementation,
// which links libX11 directly and dlopens the extensions.
//
// Convention: pointers passed IN to Xlib that reference Go memory are typed
// pointers or unsafe.Pointer (purego keeps them alive for the call);
// pointers received FROM Xlib are C-owned memory represented as uintptr and
// must be freed with xFree where the Xlib contract requires it.
//
// Deviations from the C sources:
//   - X_HAVE_UTF8_STRING is assumed: every libX11.so.6 provides the Xutf8
//     functions, so the Xmb/Xwc locale fallbacks are not bound.
//   - XSaveContext/XFindContext/XDeleteContext/XUniqueContext are not bound;
//     the XID-to-window mapping uses a Go map instead, since Go pointers
//     must not be stored in C memory.

var libX11 uintptr

var (
	// Function forms of the Xlib display and screen accessor macros
	// (DefaultScreen, RootWindow, ...).
	xConnectionNumber func(display uintptr) int32
	xDefaultDepth     func(display uintptr, screen int32) int32
	xDefaultScreen    func(display uintptr) int32
	xDefaultVisual    func(display uintptr, screen int32) uintptr
	xDisplayHeight    func(display uintptr, screen int32) int32
	xDisplayHeightMM  func(display uintptr, screen int32) int32
	xDisplayWidth     func(display uintptr, screen int32) int32
	xDisplayWidthMM   func(display uintptr, screen int32) int32
	xRootWindow       func(display uintptr, screen int32) XID

	xAllocClassHint            func() uintptr
	xAllocSizeHints            func() uintptr
	xAllocWMHints              func() uintptr
	xChangeProperty            func(display uintptr, w XID, property Atom, typ Atom, format int32, mode int32, data unsafe.Pointer, nelements int32) int32
	xChangeWindowAttributes    func(display uintptr, w XID, valuemask _Culong, attributes *XSetWindowAttributes) int32
	xCheckIfEvent              func(display uintptr, event *XEvent, predicate uintptr, arg uintptr) bool
	xCheckTypedWindowEvent     func(display uintptr, w XID, eventType int32, event *XEvent) bool
	xCloseDisplay              func(display uintptr) int32
	xCloseIM                   func(im uintptr) int32
	xConvertSelection          func(display uintptr, selection Atom, target Atom, property Atom, requestor XID, time Time) int32
	xCreateColormap            func(display uintptr, w XID, visual uintptr, alloc int32) XID
	xCreateFontCursor          func(display uintptr, shape uint32) XID
	xCreateRegion              func() Region
	xCreateWindow              func(display uintptr, parent XID, x, y int32, width, height, borderWidth uint32, depth int32, class uint32, visual uintptr, valuemask _Culong, attributes *XSetWindowAttributes) XID
	xDefineCursor              func(display uintptr, w XID, cursor XID) int32
	xDeleteProperty            func(display uintptr, w XID, property Atom) int32
	xDestroyIC                 func(ic uintptr)
	xDestroyRegion             func(r Region) int32
	xDestroyWindow             func(display uintptr, w XID) int32
	xDisplayKeycodes           func(display uintptr, minKeycodes, maxKeycodes *int32) int32
	xEventsQueued              func(display uintptr, mode int32) int32
	xFilterEvent               func(event *XEvent, w XID) bool
	xFlush                     func(display uintptr) int32
	xFree                      func(data uintptr) int32
	xFreeColormap              func(display uintptr, colormap XID) int32
	xFreeCursor                func(display uintptr, cursor XID) int32
	xFreeEventData             func(display uintptr, cookie *XGenericEventCookie)
	xGetErrorText              func(display uintptr, code int32, buffer []byte, length int32) int32
	xGetEventData              func(display uintptr, cookie *XGenericEventCookie) bool
	xGetICValues               func(ic uintptr, key string, value *_Culong, term uintptr) uintptr
	xGetIMValues               func(im uintptr, key string, value *uintptr, term uintptr) uintptr
	xGetInputFocus             func(display uintptr, focusReturn *XID, revertToReturn *int32) int32
	xGetKeyboardMapping        func(display uintptr, firstKeycode KeyCode, keycodeCount int32, keysymsPerKeycodeReturn *int32) uintptr
	xGetScreenSaver            func(display uintptr, timeout, interval, preferBlanking, allowExposures *int32) int32
	xGetSelectionOwner         func(display uintptr, selection Atom) XID
	xGetVisualInfo             func(display uintptr, vinfoMask _Clong, vinfoTemplate *XVisualInfo, nitemsReturn *int32) uintptr
	xGetWMNormalHints          func(display uintptr, w XID, hints *XSizeHints, supplied *_Clong) int32
	xGetWindowAttributes       func(display uintptr, w XID, attributes *XWindowAttributes) int32
	xGetWindowProperty         func(display uintptr, w XID, property Atom, longOffset, longLength _Clong, delete bool, reqType Atom, actualTypeReturn *Atom, actualFormatReturn *int32, nitemsReturn *_Culong, bytesAfterReturn *_Culong, propReturn *uintptr) int32
	xGrabPointer               func(display uintptr, grabWindow XID, ownerEvents bool, eventMask uint32, pointerMode, keyboardMode int32, confineTo XID, cursor XID, time Time) int32
	xIconifyWindow             func(display uintptr, w XID, screen int32) int32
	xInitThreads               func() int32
	xInternAtom                func(display uintptr, atomName string, onlyIfExists bool) Atom
	xLookupString              func(eventStruct *XKeyEvent, bufferReturn []byte, bytesBuffer int32, keysymReturn *KeySym, statusInOut uintptr) int32
	xMapRaised                 func(display uintptr, w XID) int32
	xMapWindow                 func(display uintptr, w XID) int32
	xMoveResizeWindow          func(display uintptr, w XID, x, y int32, width, height uint32) int32
	xMoveWindow                func(display uintptr, w XID, x, y int32) int32
	xNextEvent                 func(display uintptr, eventReturn *XEvent) int32
	xOpenDisplay               func(displayName uintptr) uintptr
	xOpenIM                    func(display uintptr, db uintptr, resName uintptr, resClass uintptr) uintptr
	xPeekEvent                 func(display uintptr, eventReturn *XEvent) int32
	xPending                   func(display uintptr) int32
	xQLength                   func(display uintptr) int32
	xQueryExtension            func(display uintptr, name string, majorOpcodeReturn, firstEventReturn, firstErrorReturn *int32) bool
	xQueryPointer              func(display uintptr, w XID, rootReturn, childReturn *XID, rootXReturn, rootYReturn, winXReturn, winYReturn *int32, maskReturn *uint32) bool
	xRaiseWindow               func(display uintptr, w XID) int32
	xResizeWindow              func(display uintptr, w XID, width, height uint32) int32
	xResourceManagerString     func(display uintptr) uintptr
	xSelectInput               func(display uintptr, w XID, eventMask _Clong) int32
	xSendEvent                 func(display uintptr, w XID, propagate bool, eventMask _Clong, eventSend *XEvent) int32
	xSetClassHint              func(display uintptr, w XID, classHints *XClassHint) int32
	xSetErrorHandler           func(handler uintptr) uintptr
	xSetICFocus                func(ic uintptr)
	xSetInputFocus             func(display uintptr, focus XID, revertTo int32, time Time) int32
	xSetLocaleModifiers        func(modifierList string) uintptr
	xSetScreenSaver            func(display uintptr, timeout, interval, preferBlanking, allowExposures int32) int32
	xSetSelectionOwner         func(display uintptr, selection Atom, owner XID, time Time) int32
	xSetWMHints                func(display uintptr, w XID, wmHints *XWMHints) int32
	xSetWMNormalHints          func(display uintptr, w XID, hints *XSizeHints)
	xSetWMProtocols            func(display uintptr, w XID, protocols *Atom, count int32) int32
	xSupportsLocale            func() bool
	xSync                      func(display uintptr, discard bool) int32
	xTranslateCoordinates      func(display uintptr, srcW, destW XID, srcX, srcY int32, destXReturn, destYReturn *int32, childReturn *XID) bool
	xUndefineCursor            func(display uintptr, w XID) int32
	xUngrabPointer             func(display uintptr, time Time) int32
	xUnmapWindow               func(display uintptr, w XID) int32
	xUnsetICFocus              func(ic uintptr)
	xWarpPointer               func(display uintptr, srcW, destW XID, srcX, srcY int32, srcWidth, srcHeight uint32, destX, destY int32) int32
	xkbFreeKeyboard            func(xkb uintptr, which uint32, freeDesc bool)
	xkbFreeNames               func(xkb uintptr, which uint32, freeMap bool)
	xkbGetMap                  func(display uintptr, which uint32, deviceSpec uint32) uintptr
	xkbGetNames                func(display uintptr, which uint32, xkb uintptr) int32
	xkbGetState                func(display uintptr, deviceSpec uint32, stateReturn *XkbStateRec) int32
	xkbKeycodeToKeysym         func(display uintptr, kc uint32, group, level int32) KeySym
	xkbQueryExtension          func(display uintptr, opcodeReturn, eventBaseReturn, errorBaseReturn, majorReturn, minorReturn *int32) bool
	xkbSelectEventDetails      func(display uintptr, deviceSpec uint32, eventType uint32, affect, details _Culong) bool
	xkbSetDetectableAutoRepeat func(display uintptr, detectable bool, supportedReturn *int32) bool
	xrmDestroyDatabase         func(database uintptr)
	xrmGetResource             func(database uintptr, strName, strClass string, strTypeReturn *uintptr, valueReturn *XrmValue) bool
	xrmGetStringDatabase       func(data uintptr) uintptr
	xrmInitialize              func()
	xutf8LookupString          func(ic uintptr, event *XKeyEvent, bufferReturn []byte, bytesBuffer int32, keysymReturn *KeySym, statusReturn *int32) int32
	xutf8SetWMProperties       func(display uintptr, w XID, windowName, iconName string, argv uintptr, argc int32, normalHints, wmHints, classHints uintptr)

	// Variadic in C; bound with the exact argument shape used by this
	// package (XNInputStyle, XNClientWindow, XNFocusWindow, NULL).
	xCreateIC func(im uintptr, k1 string, v1 XIMStyle, k2 string, v2 XID, k3 string, v3 XID, term uintptr) uintptr

	// setlocale from libc, resolved via RTLD_DEFAULT so that the libc soname
	// (glibc vs musl) does not matter. setlocaleQuery is the same function
	// with a pointer-typed locale argument, for passing NULL to query the
	// current locale.
	setlocale      func(category int32, locale string) uintptr
	setlocaleQuery func(category int32, locale uintptr) uintptr
)

// XrmValue is the Xrm resource value struct.
type XrmValue struct {
	Size uint32
	Addr uintptr
}

// openX11Library dlopens the first available of the given sonames.
func openX11Library(names ...string) (uintptr, error) {
	var errs []error
	for _, name := range names {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			return lib, nil
		}
		errs = append(errs, err)
	}
	return 0, fmt.Errorf("glfw: failed to load any of %v: %w", names, errs[0])
}

// initLibX11 loads libX11 and registers its functions. It must be called
// before any other function in this file.
func initLibX11() error {
	if libX11 != 0 {
		return nil
	}
	lib, err := openX11Library("libX11.so.6", "libX11.so")
	if err != nil {
		return err
	}
	libX11 = lib

	purego.RegisterLibFunc(&xConnectionNumber, lib, "XConnectionNumber")
	purego.RegisterLibFunc(&xDefaultDepth, lib, "XDefaultDepth")
	purego.RegisterLibFunc(&xDefaultScreen, lib, "XDefaultScreen")
	purego.RegisterLibFunc(&xDefaultVisual, lib, "XDefaultVisual")
	purego.RegisterLibFunc(&xDisplayHeight, lib, "XDisplayHeight")
	purego.RegisterLibFunc(&xDisplayHeightMM, lib, "XDisplayHeightMM")
	purego.RegisterLibFunc(&xDisplayWidth, lib, "XDisplayWidth")
	purego.RegisterLibFunc(&xDisplayWidthMM, lib, "XDisplayWidthMM")
	purego.RegisterLibFunc(&xRootWindow, lib, "XRootWindow")

	purego.RegisterLibFunc(&xAllocClassHint, lib, "XAllocClassHint")
	purego.RegisterLibFunc(&xAllocSizeHints, lib, "XAllocSizeHints")
	purego.RegisterLibFunc(&xAllocWMHints, lib, "XAllocWMHints")
	purego.RegisterLibFunc(&xChangeProperty, lib, "XChangeProperty")
	purego.RegisterLibFunc(&xChangeWindowAttributes, lib, "XChangeWindowAttributes")
	purego.RegisterLibFunc(&xCheckIfEvent, lib, "XCheckIfEvent")
	purego.RegisterLibFunc(&xCheckTypedWindowEvent, lib, "XCheckTypedWindowEvent")
	purego.RegisterLibFunc(&xCloseDisplay, lib, "XCloseDisplay")
	purego.RegisterLibFunc(&xCloseIM, lib, "XCloseIM")
	purego.RegisterLibFunc(&xConvertSelection, lib, "XConvertSelection")
	purego.RegisterLibFunc(&xCreateColormap, lib, "XCreateColormap")
	purego.RegisterLibFunc(&xCreateFontCursor, lib, "XCreateFontCursor")
	purego.RegisterLibFunc(&xCreateIC, lib, "XCreateIC")
	purego.RegisterLibFunc(&xCreateRegion, lib, "XCreateRegion")
	purego.RegisterLibFunc(&xCreateWindow, lib, "XCreateWindow")
	purego.RegisterLibFunc(&xDefineCursor, lib, "XDefineCursor")
	purego.RegisterLibFunc(&xDeleteProperty, lib, "XDeleteProperty")
	purego.RegisterLibFunc(&xDestroyIC, lib, "XDestroyIC")
	purego.RegisterLibFunc(&xDestroyRegion, lib, "XDestroyRegion")
	purego.RegisterLibFunc(&xDestroyWindow, lib, "XDestroyWindow")
	purego.RegisterLibFunc(&xDisplayKeycodes, lib, "XDisplayKeycodes")
	purego.RegisterLibFunc(&xEventsQueued, lib, "XEventsQueued")
	purego.RegisterLibFunc(&xFilterEvent, lib, "XFilterEvent")
	purego.RegisterLibFunc(&xFlush, lib, "XFlush")
	purego.RegisterLibFunc(&xFree, lib, "XFree")
	purego.RegisterLibFunc(&xFreeColormap, lib, "XFreeColormap")
	purego.RegisterLibFunc(&xFreeCursor, lib, "XFreeCursor")
	purego.RegisterLibFunc(&xFreeEventData, lib, "XFreeEventData")
	purego.RegisterLibFunc(&xGetErrorText, lib, "XGetErrorText")
	purego.RegisterLibFunc(&xGetEventData, lib, "XGetEventData")
	purego.RegisterLibFunc(&xGetICValues, lib, "XGetICValues")
	purego.RegisterLibFunc(&xGetIMValues, lib, "XGetIMValues")
	purego.RegisterLibFunc(&xGetInputFocus, lib, "XGetInputFocus")
	purego.RegisterLibFunc(&xGetKeyboardMapping, lib, "XGetKeyboardMapping")
	purego.RegisterLibFunc(&xGetScreenSaver, lib, "XGetScreenSaver")
	purego.RegisterLibFunc(&xGetSelectionOwner, lib, "XGetSelectionOwner")
	purego.RegisterLibFunc(&xGetVisualInfo, lib, "XGetVisualInfo")
	purego.RegisterLibFunc(&xGetWMNormalHints, lib, "XGetWMNormalHints")
	purego.RegisterLibFunc(&xGetWindowAttributes, lib, "XGetWindowAttributes")
	purego.RegisterLibFunc(&xGetWindowProperty, lib, "XGetWindowProperty")
	purego.RegisterLibFunc(&xGrabPointer, lib, "XGrabPointer")
	purego.RegisterLibFunc(&xIconifyWindow, lib, "XIconifyWindow")
	purego.RegisterLibFunc(&xInitThreads, lib, "XInitThreads")
	purego.RegisterLibFunc(&xInternAtom, lib, "XInternAtom")
	purego.RegisterLibFunc(&xLookupString, lib, "XLookupString")
	purego.RegisterLibFunc(&xMapRaised, lib, "XMapRaised")
	purego.RegisterLibFunc(&xMapWindow, lib, "XMapWindow")
	purego.RegisterLibFunc(&xMoveResizeWindow, lib, "XMoveResizeWindow")
	purego.RegisterLibFunc(&xMoveWindow, lib, "XMoveWindow")
	purego.RegisterLibFunc(&xNextEvent, lib, "XNextEvent")
	purego.RegisterLibFunc(&xOpenDisplay, lib, "XOpenDisplay")
	purego.RegisterLibFunc(&xOpenIM, lib, "XOpenIM")
	purego.RegisterLibFunc(&xPeekEvent, lib, "XPeekEvent")
	purego.RegisterLibFunc(&xPending, lib, "XPending")
	purego.RegisterLibFunc(&xQLength, lib, "XQLength")
	purego.RegisterLibFunc(&xQueryExtension, lib, "XQueryExtension")
	purego.RegisterLibFunc(&xQueryPointer, lib, "XQueryPointer")
	purego.RegisterLibFunc(&xRaiseWindow, lib, "XRaiseWindow")
	purego.RegisterLibFunc(&xResizeWindow, lib, "XResizeWindow")
	purego.RegisterLibFunc(&xResourceManagerString, lib, "XResourceManagerString")
	purego.RegisterLibFunc(&xSelectInput, lib, "XSelectInput")
	purego.RegisterLibFunc(&xSendEvent, lib, "XSendEvent")
	purego.RegisterLibFunc(&xSetClassHint, lib, "XSetClassHint")
	purego.RegisterLibFunc(&xSetErrorHandler, lib, "XSetErrorHandler")
	purego.RegisterLibFunc(&xSetICFocus, lib, "XSetICFocus")
	purego.RegisterLibFunc(&xSetInputFocus, lib, "XSetInputFocus")
	purego.RegisterLibFunc(&xSetLocaleModifiers, lib, "XSetLocaleModifiers")
	purego.RegisterLibFunc(&xSetScreenSaver, lib, "XSetScreenSaver")
	purego.RegisterLibFunc(&xSetSelectionOwner, lib, "XSetSelectionOwner")
	purego.RegisterLibFunc(&xSetWMHints, lib, "XSetWMHints")
	purego.RegisterLibFunc(&xSetWMNormalHints, lib, "XSetWMNormalHints")
	purego.RegisterLibFunc(&xSetWMProtocols, lib, "XSetWMProtocols")
	purego.RegisterLibFunc(&xSupportsLocale, lib, "XSupportsLocale")
	purego.RegisterLibFunc(&xSync, lib, "XSync")
	purego.RegisterLibFunc(&xTranslateCoordinates, lib, "XTranslateCoordinates")
	purego.RegisterLibFunc(&xUndefineCursor, lib, "XUndefineCursor")
	purego.RegisterLibFunc(&xUngrabPointer, lib, "XUngrabPointer")
	purego.RegisterLibFunc(&xUnmapWindow, lib, "XUnmapWindow")
	purego.RegisterLibFunc(&xUnsetICFocus, lib, "XUnsetICFocus")
	purego.RegisterLibFunc(&xWarpPointer, lib, "XWarpPointer")
	purego.RegisterLibFunc(&xkbFreeKeyboard, lib, "XkbFreeKeyboard")
	purego.RegisterLibFunc(&xkbFreeNames, lib, "XkbFreeNames")
	purego.RegisterLibFunc(&xkbGetMap, lib, "XkbGetMap")
	purego.RegisterLibFunc(&xkbGetNames, lib, "XkbGetNames")
	purego.RegisterLibFunc(&xkbGetState, lib, "XkbGetState")
	purego.RegisterLibFunc(&xkbKeycodeToKeysym, lib, "XkbKeycodeToKeysym")
	purego.RegisterLibFunc(&xkbQueryExtension, lib, "XkbQueryExtension")
	purego.RegisterLibFunc(&xkbSelectEventDetails, lib, "XkbSelectEventDetails")
	purego.RegisterLibFunc(&xkbSetDetectableAutoRepeat, lib, "XkbSetDetectableAutoRepeat")
	purego.RegisterLibFunc(&xrmDestroyDatabase, lib, "XrmDestroyDatabase")
	purego.RegisterLibFunc(&xrmGetResource, lib, "XrmGetResource")
	purego.RegisterLibFunc(&xrmGetStringDatabase, lib, "XrmGetStringDatabase")
	purego.RegisterLibFunc(&xrmInitialize, lib, "XrmInitialize")
	purego.RegisterLibFunc(&xutf8LookupString, lib, "Xutf8LookupString")
	purego.RegisterLibFunc(&xutf8SetWMProperties, lib, "Xutf8SetWMProperties")

	setlocaleSym, err := purego.Dlsym(purego.RTLD_DEFAULT, "setlocale")
	if err != nil {
		return err
	}
	purego.RegisterFunc(&setlocale, setlocaleSym)
	purego.RegisterFunc(&setlocaleQuery, setlocaleSym)

	return nil
}

// goString copies a NUL-terminated C string.
func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	var n int
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}

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

//go:build (freebsd || linux || netbsd) && cgo && ebitenginexlibcheck

package xlibcheck

// The sizes and offsets exported here are the values the C compiler computes
// for the Xlib structs on the target data model. The external test compares
// them against internal/glfw's Go mirrors. cgo lives in this non-test file
// because cgo is not allowed in an external (_test) test package.

/*
#cgo pkg-config: x11 xrandr xinerama xi xrender xcursor

#include <stddef.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/XKBlib.h>
#include <X11/extensions/Xrandr.h>
#include <X11/extensions/Xinerama.h>
#include <X11/extensions/Xrender.h>
#include <X11/Xcursor/Xcursor.h>
#include <X11/extensions/XInput2.h>

enum {
	goff_XkbStateNotifyEvent_xkb_type = offsetof(XkbStateNotifyEvent, xkb_type),
	goff_XkbStateNotifyEvent_changed  = offsetof(XkbStateNotifyEvent, changed),
	goff_XkbStateNotifyEvent_group    = offsetof(XkbStateNotifyEvent, group),
};
*/
import "C"

// Sizeof* report sizeof of the corresponding Xlib struct.
var (
	SizeofXEvent                 = uintptr(C.sizeof_XEvent)
	SizeofXAnyEvent              = uintptr(C.sizeof_XAnyEvent)
	SizeofXKeyEvent              = uintptr(C.sizeof_XKeyEvent)
	SizeofXButtonEvent           = uintptr(C.sizeof_XButtonEvent)
	SizeofXMotionEvent           = uintptr(C.sizeof_XMotionEvent)
	SizeofXCrossingEvent         = uintptr(C.sizeof_XCrossingEvent)
	SizeofXFocusChangeEvent      = uintptr(C.sizeof_XFocusChangeEvent)
	SizeofXExposeEvent           = uintptr(C.sizeof_XExposeEvent)
	SizeofXVisibilityEvent       = uintptr(C.sizeof_XVisibilityEvent)
	SizeofXDestroyWindowEvent    = uintptr(C.sizeof_XDestroyWindowEvent)
	SizeofXUnmapEvent            = uintptr(C.sizeof_XUnmapEvent)
	SizeofXMapEvent              = uintptr(C.sizeof_XMapEvent)
	SizeofXReparentEvent         = uintptr(C.sizeof_XReparentEvent)
	SizeofXConfigureEvent        = uintptr(C.sizeof_XConfigureEvent)
	SizeofXPropertyEvent         = uintptr(C.sizeof_XPropertyEvent)
	SizeofXSelectionRequestEvent = uintptr(C.sizeof_XSelectionRequestEvent)
	SizeofXSelectionEvent        = uintptr(C.sizeof_XSelectionEvent)
	SizeofXSelectionClearEvent   = uintptr(C.sizeof_XSelectionClearEvent)
	SizeofXClientMessageEvent    = uintptr(C.sizeof_XClientMessageEvent)
	SizeofXGenericEventCookie    = uintptr(C.sizeof_XGenericEventCookie)
	SizeofXErrorEvent            = uintptr(C.sizeof_XErrorEvent)
	SizeofXSetWindowAttributes   = uintptr(C.sizeof_XSetWindowAttributes)
	SizeofXWindowAttributes      = uintptr(C.sizeof_XWindowAttributes)
	SizeofVisual                 = uintptr(C.sizeof_Visual)
	SizeofXVisualInfo            = uintptr(C.sizeof_XVisualInfo)
	SizeofXSizeHints             = uintptr(C.sizeof_XSizeHints)
	SizeofXWMHints               = uintptr(C.sizeof_XWMHints)
	SizeofXClassHint             = uintptr(C.sizeof_XClassHint)
	SizeofXIMStyles              = uintptr(C.sizeof_XIMStyles)
	SizeofXRectangle             = uintptr(C.sizeof_XRectangle)
	SizeofXkbDescRec             = uintptr(C.sizeof_XkbDescRec)
	SizeofXkbNamesRec            = uintptr(C.sizeof_XkbNamesRec)
	SizeofXkbKeyNameRec          = uintptr(C.sizeof_XkbKeyNameRec)
	SizeofXkbKeyAliasRec         = uintptr(C.sizeof_XkbKeyAliasRec)
	SizeofXkbStateRec            = uintptr(C.sizeof_XkbStateRec)
	SizeofXkbAnyEvent            = uintptr(C.sizeof_XkbAnyEvent)
	SizeofXkbStateNotifyEvent    = uintptr(C.sizeof_XkbStateNotifyEvent)
	SizeofXRenderDirectFormat    = uintptr(C.sizeof_XRenderDirectFormat)
	SizeofXRenderPictFormat      = uintptr(C.sizeof_XRenderPictFormat)
	SizeofXRRModeInfo            = uintptr(C.sizeof_XRRModeInfo)
	SizeofXRRScreenResources     = uintptr(C.sizeof_XRRScreenResources)
	SizeofXRROutputInfo          = uintptr(C.sizeof_XRROutputInfo)
	SizeofXRRCrtcInfo            = uintptr(C.sizeof_XRRCrtcInfo)
	SizeofXineramaScreenInfo     = uintptr(C.sizeof_XineramaScreenInfo)
	SizeofXcursorImage           = uintptr(C.sizeof_XcursorImage)
	SizeofXIEventMask            = uintptr(C.sizeof_XIEventMask)
	SizeofXIValuatorState        = uintptr(C.sizeof_XIValuatorState)
	SizeofXIRawEvent             = uintptr(C.sizeof_XIRawEvent)
)

// Offsetof* report offsetof of the read fields of XkbStateNotifyEvent, whose
// total size is data-model-specific (the Go mirror collapses the unread tail).
var (
	OffsetofXkbStateNotifyEventXkbType = uintptr(C.goff_XkbStateNotifyEvent_xkb_type)
	OffsetofXkbStateNotifyEventChanged = uintptr(C.goff_XkbStateNotifyEvent_changed)
	OffsetofXkbStateNotifyEventGroup   = uintptr(C.goff_XkbStateNotifyEvent_group)
)

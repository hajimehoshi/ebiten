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

package glfw

// This file holds the C compiler's sizes and offsets for the Xlib structs. The
// cross-check in xlibcheck_linbsd_test.go compares them against this package's
// Go mirrors. cgo lives here, in a regular file, because import "C" is rejected
// in a _test.go file.
//
// The whole check is gated by the ebitenginexlibcheck build tag, so ordinary
// builds neither use cgo nor need the X11 development headers; only the
// dedicated CI step that sets the tag compiles it. The test reads these
// unexported values and the unexported mirror types directly, so nothing has to
// be exported.

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

// sizeof* report sizeof of the corresponding Xlib struct.
var (
	sizeofXEvent                 = uintptr(C.sizeof_XEvent)
	sizeofXAnyEvent              = uintptr(C.sizeof_XAnyEvent)
	sizeofXKeyEvent              = uintptr(C.sizeof_XKeyEvent)
	sizeofXButtonEvent           = uintptr(C.sizeof_XButtonEvent)
	sizeofXMotionEvent           = uintptr(C.sizeof_XMotionEvent)
	sizeofXCrossingEvent         = uintptr(C.sizeof_XCrossingEvent)
	sizeofXFocusChangeEvent      = uintptr(C.sizeof_XFocusChangeEvent)
	sizeofXExposeEvent           = uintptr(C.sizeof_XExposeEvent)
	sizeofXVisibilityEvent       = uintptr(C.sizeof_XVisibilityEvent)
	sizeofXDestroyWindowEvent    = uintptr(C.sizeof_XDestroyWindowEvent)
	sizeofXUnmapEvent            = uintptr(C.sizeof_XUnmapEvent)
	sizeofXMapEvent              = uintptr(C.sizeof_XMapEvent)
	sizeofXReparentEvent         = uintptr(C.sizeof_XReparentEvent)
	sizeofXConfigureEvent        = uintptr(C.sizeof_XConfigureEvent)
	sizeofXPropertyEvent         = uintptr(C.sizeof_XPropertyEvent)
	sizeofXSelectionRequestEvent = uintptr(C.sizeof_XSelectionRequestEvent)
	sizeofXSelectionEvent        = uintptr(C.sizeof_XSelectionEvent)
	sizeofXSelectionClearEvent   = uintptr(C.sizeof_XSelectionClearEvent)
	sizeofXClientMessageEvent    = uintptr(C.sizeof_XClientMessageEvent)
	sizeofXGenericEventCookie    = uintptr(C.sizeof_XGenericEventCookie)
	sizeofXErrorEvent            = uintptr(C.sizeof_XErrorEvent)
	sizeofXSetWindowAttributes   = uintptr(C.sizeof_XSetWindowAttributes)
	sizeofXWindowAttributes      = uintptr(C.sizeof_XWindowAttributes)
	sizeofVisual                 = uintptr(C.sizeof_Visual)
	sizeofXVisualInfo            = uintptr(C.sizeof_XVisualInfo)
	sizeofXSizeHints             = uintptr(C.sizeof_XSizeHints)
	sizeofXWMHints               = uintptr(C.sizeof_XWMHints)
	sizeofXClassHint             = uintptr(C.sizeof_XClassHint)
	sizeofXIMStyles              = uintptr(C.sizeof_XIMStyles)
	sizeofXRectangle             = uintptr(C.sizeof_XRectangle)
	sizeofXkbDescRec             = uintptr(C.sizeof_XkbDescRec)
	sizeofXkbNamesRec            = uintptr(C.sizeof_XkbNamesRec)
	sizeofXkbKeyNameRec          = uintptr(C.sizeof_XkbKeyNameRec)
	sizeofXkbKeyAliasRec         = uintptr(C.sizeof_XkbKeyAliasRec)
	sizeofXkbStateRec            = uintptr(C.sizeof_XkbStateRec)
	sizeofXkbAnyEvent            = uintptr(C.sizeof_XkbAnyEvent)
	sizeofXkbStateNotifyEvent    = uintptr(C.sizeof_XkbStateNotifyEvent)
	sizeofXRenderDirectFormat    = uintptr(C.sizeof_XRenderDirectFormat)
	sizeofXRenderPictFormat      = uintptr(C.sizeof_XRenderPictFormat)
	sizeofXRRModeInfo            = uintptr(C.sizeof_XRRModeInfo)
	sizeofXRRScreenResources     = uintptr(C.sizeof_XRRScreenResources)
	sizeofXRROutputInfo          = uintptr(C.sizeof_XRROutputInfo)
	sizeofXRRCrtcInfo            = uintptr(C.sizeof_XRRCrtcInfo)
	sizeofXineramaScreenInfo     = uintptr(C.sizeof_XineramaScreenInfo)
	sizeofXcursorImage           = uintptr(C.sizeof_XcursorImage)
	sizeofXIEventMask            = uintptr(C.sizeof_XIEventMask)
	sizeofXIValuatorState        = uintptr(C.sizeof_XIValuatorState)
	sizeofXIRawEvent             = uintptr(C.sizeof_XIRawEvent)
)

// offsetof* report offsetof of the read fields of XkbStateNotifyEvent, whose
// total size is data-model-specific (the Go mirror collapses the unread tail).
var (
	offsetofXkbStateNotifyEventXkbType = uintptr(C.goff_XkbStateNotifyEvent_xkb_type)
	offsetofXkbStateNotifyEventChanged = uintptr(C.goff_XkbStateNotifyEvent_changed)
	offsetofXkbStateNotifyEventGroup   = uintptr(C.goff_XkbStateNotifyEvent_group)
)

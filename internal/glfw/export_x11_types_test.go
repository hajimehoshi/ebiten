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

// These aliases re-export the unexported X11 struct types for the golden test
// in x11_types_linbsd_test.go, which is an external (glfw_test) test and so
// cannot name the unexported types directly.

type (
	XEvent                 = _XEvent
	XAnyEvent              = _XAnyEvent
	XKeyEvent              = _XKeyEvent
	XButtonEvent           = _XButtonEvent
	XMotionEvent           = _XMotionEvent
	XCrossingEvent         = _XCrossingEvent
	XFocusChangeEvent      = _XFocusChangeEvent
	XExposeEvent           = _XExposeEvent
	XVisibilityEvent       = _XVisibilityEvent
	XDestroyWindowEvent    = _XDestroyWindowEvent
	XUnmapEvent            = _XUnmapEvent
	XMapEvent              = _XMapEvent
	XReparentEvent         = _XReparentEvent
	XConfigureEvent        = _XConfigureEvent
	XPropertyEvent         = _XPropertyEvent
	XSelectionRequestEvent = _XSelectionRequestEvent
	XSelectionEvent        = _XSelectionEvent
	XSelectionClearEvent   = _XSelectionClearEvent
	XClientMessageEvent    = _XClientMessageEvent
	XGenericEventCookie    = _XGenericEventCookie
	XErrorEvent            = _XErrorEvent
	XSetWindowAttributes   = _XSetWindowAttributes
	XWindowAttributes      = _XWindowAttributes
	Visual                 = _Visual
	XVisualInfo            = _XVisualInfo
	XSizeHints             = _XSizeHints
	XWMHints               = _XWMHints
	XClassHint             = _XClassHint
	XIMStyles              = _XIMStyles
	XRectangle             = _XRectangle
	XkbDescRec             = _XkbDescRec
	XkbNamesRec            = _XkbNamesRec
	XkbKeyNameRec          = _XkbKeyNameRec
	XkbKeyAliasRec         = _XkbKeyAliasRec
	XkbStateRec            = _XkbStateRec
	XkbAnyEvent            = _XkbAnyEvent
	XkbStateNotifyEvent    = _XkbStateNotifyEvent
	XRenderDirectFormat    = _XRenderDirectFormat
	XRenderPictFormat      = _XRenderPictFormat
	XRRModeInfo            = _XRRModeInfo
	XRRScreenResources     = _XRRScreenResources
	XRROutputInfo          = _XRROutputInfo
	XRRCrtcInfo            = _XRRCrtcInfo
	XineramaScreenInfo     = _XineramaScreenInfo
	XcursorImage           = _XcursorImage
	XIEventMask            = _XIEventMask
	XIValuatorState        = _XIValuatorState
	XIRawEvent             = _XIRawEvent
	XSyncValue             = _XSyncValue
)

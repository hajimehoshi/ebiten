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

// This test cross-checks the Go mirrors of the Xlib structs against the real C
// headers, so it works on whatever data model the C compiler targets. Run it
// with a 32-bit toolchain (e.g. GOARCH=386 with a multilib/i386 cgo setup) to
// verify the ILP32 layout, which the pure-Go golden test in
// x11_types_linbsd_test.go cannot cover (its goldens are LP64-only).
//
// The C reference values come from xlibcheck_linbsd.go (cgo lives there, not
// here, because import "C" is not allowed in a _test.go file). This test is a
// white-box (package glfw) test so it can read those unexported values and the
// unexported mirror types directly, without exporting anything.
//
// Struct sizes catch padding/alignment mistakes on every data model. The field
// offsets are pinned for LP64 by the golden test and, for the mirrors that spell
// out every field with word-sized types, follow from the size match on ILP32
// too. XkbStateNotifyEvent is the exception: it is a view into the XEvent union
// with a collapsed opaque tail, so only the offsets actually read are checked
// (its total size is LP64-specific).

package glfw

import (
	"math/bits"
	"testing"
	"unsafe"
)

func TestXlibStructSizes(t *testing.T) {
	checks := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"XEvent", unsafe.Sizeof(_XEvent{}), sizeofXEvent},
		{"XAnyEvent", unsafe.Sizeof(_XAnyEvent{}), sizeofXAnyEvent},
		{"XKeyEvent", unsafe.Sizeof(_XKeyEvent{}), sizeofXKeyEvent},
		{"XButtonEvent", unsafe.Sizeof(_XButtonEvent{}), sizeofXButtonEvent},
		{"XMotionEvent", unsafe.Sizeof(_XMotionEvent{}), sizeofXMotionEvent},
		{"XCrossingEvent", unsafe.Sizeof(_XCrossingEvent{}), sizeofXCrossingEvent},
		{"XFocusChangeEvent", unsafe.Sizeof(_XFocusChangeEvent{}), sizeofXFocusChangeEvent},
		{"XExposeEvent", unsafe.Sizeof(_XExposeEvent{}), sizeofXExposeEvent},
		{"XVisibilityEvent", unsafe.Sizeof(_XVisibilityEvent{}), sizeofXVisibilityEvent},
		{"XDestroyWindowEvent", unsafe.Sizeof(_XDestroyWindowEvent{}), sizeofXDestroyWindowEvent},
		{"XUnmapEvent", unsafe.Sizeof(_XUnmapEvent{}), sizeofXUnmapEvent},
		{"XMapEvent", unsafe.Sizeof(_XMapEvent{}), sizeofXMapEvent},
		{"XReparentEvent", unsafe.Sizeof(_XReparentEvent{}), sizeofXReparentEvent},
		{"XConfigureEvent", unsafe.Sizeof(_XConfigureEvent{}), sizeofXConfigureEvent},
		{"XPropertyEvent", unsafe.Sizeof(_XPropertyEvent{}), sizeofXPropertyEvent},
		{"XSelectionRequestEvent", unsafe.Sizeof(_XSelectionRequestEvent{}), sizeofXSelectionRequestEvent},
		{"XSelectionEvent", unsafe.Sizeof(_XSelectionEvent{}), sizeofXSelectionEvent},
		{"XSelectionClearEvent", unsafe.Sizeof(_XSelectionClearEvent{}), sizeofXSelectionClearEvent},
		{"XClientMessageEvent", unsafe.Sizeof(_XClientMessageEvent{}), sizeofXClientMessageEvent},
		{"XGenericEventCookie", unsafe.Sizeof(_XGenericEventCookie{}), sizeofXGenericEventCookie},
		{"XErrorEvent", unsafe.Sizeof(_XErrorEvent{}), sizeofXErrorEvent},
		{"XSetWindowAttributes", unsafe.Sizeof(_XSetWindowAttributes{}), sizeofXSetWindowAttributes},
		{"XWindowAttributes", unsafe.Sizeof(_XWindowAttributes{}), sizeofXWindowAttributes},
		{"Visual", unsafe.Sizeof(_Visual{}), sizeofVisual},
		{"XVisualInfo", unsafe.Sizeof(_XVisualInfo{}), sizeofXVisualInfo},
		{"XSizeHints", unsafe.Sizeof(_XSizeHints{}), sizeofXSizeHints},
		{"XWMHints", unsafe.Sizeof(_XWMHints{}), sizeofXWMHints},
		{"XClassHint", unsafe.Sizeof(_XClassHint{}), sizeofXClassHint},
		{"XIMStyles", unsafe.Sizeof(_XIMStyles{}), sizeofXIMStyles},
		{"XRectangle", unsafe.Sizeof(_XRectangle{}), sizeofXRectangle},
		{"XkbDescRec", unsafe.Sizeof(_XkbDescRec{}), sizeofXkbDescRec},
		{"XkbNamesRec", unsafe.Sizeof(_XkbNamesRec{}), sizeofXkbNamesRec},
		{"XkbKeyNameRec", unsafe.Sizeof(_XkbKeyNameRec{}), sizeofXkbKeyNameRec},
		{"XkbKeyAliasRec", unsafe.Sizeof(_XkbKeyAliasRec{}), sizeofXkbKeyAliasRec},
		{"XkbStateRec", unsafe.Sizeof(_XkbStateRec{}), sizeofXkbStateRec},
		{"XkbAnyEvent", unsafe.Sizeof(_XkbAnyEvent{}), sizeofXkbAnyEvent},
		{"XRenderDirectFormat", unsafe.Sizeof(_XRenderDirectFormat{}), sizeofXRenderDirectFormat},
		{"XRenderPictFormat", unsafe.Sizeof(_XRenderPictFormat{}), sizeofXRenderPictFormat},
		{"XRRModeInfo", unsafe.Sizeof(_XRRModeInfo{}), sizeofXRRModeInfo},
		{"XRRScreenResources", unsafe.Sizeof(_XRRScreenResources{}), sizeofXRRScreenResources},
		{"XRROutputInfo", unsafe.Sizeof(_XRROutputInfo{}), sizeofXRROutputInfo},
		{"XRRCrtcInfo", unsafe.Sizeof(_XRRCrtcInfo{}), sizeofXRRCrtcInfo},
		{"XineramaScreenInfo", unsafe.Sizeof(_XineramaScreenInfo{}), sizeofXineramaScreenInfo},
		{"XcursorImage", unsafe.Sizeof(_XcursorImage{}), sizeofXcursorImage},
		{"XIEventMask", unsafe.Sizeof(_XIEventMask{}), sizeofXIEventMask},
		{"XIValuatorState", unsafe.Sizeof(_XIValuatorState{}), sizeofXIValuatorState},
		{"XIRawEvent", unsafe.Sizeof(_XIRawEvent{}), sizeofXIRawEvent},
	}

	// XkbStateNotifyEvent is a view into the XEvent union; its Go mirror
	// collapses the unread tail into a fixed pad whose end alignment differs
	// between data models, so its total size is only meaningful on LP64.
	if bits.UintSize == 64 {
		checks = append(checks, struct {
			name string
			got  uintptr
			want uintptr
		}{"XkbStateNotifyEvent", unsafe.Sizeof(_XkbStateNotifyEvent{}), sizeofXkbStateNotifyEvent})
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("sizeof %s: Go %d, C %d", c.name, c.got, c.want)
		}
	}
}

func TestXkbStateNotifyEventOffsets(t *testing.T) {
	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"XkbType", unsafe.Offsetof(_XkbStateNotifyEvent{}.XkbType), offsetofXkbStateNotifyEventXkbType},
		{"Changed", unsafe.Offsetof(_XkbStateNotifyEvent{}.Changed), offsetofXkbStateNotifyEventChanged},
		{"Group", unsafe.Offsetof(_XkbStateNotifyEvent{}.Group), offsetofXkbStateNotifyEventGroup},
	} {
		if c.got != c.want {
			t.Errorf("offsetof XkbStateNotifyEvent.%s: Go %d, C %d", c.name, c.got, c.want)
		}
	}
}

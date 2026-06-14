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

// This test cross-checks internal/glfw's Go mirrors of the Xlib structs against
// the real C headers, so it works on whatever data model the C compiler
// targets. Run it with a 32-bit toolchain (e.g. GOARCH=386 with a multilib/i386
// cgo setup) to verify the ILP32 layout, which the pure-Go golden test in
// internal/glfw cannot cover (its goldens are LP64-only).
//
// The C reference values come from the xlibcheck package (cgo lives there, not
// here, because cgo is not allowed in an external test package).
//
// Struct sizes catch padding/alignment mistakes on every data model. The field
// offsets are pinned for LP64 by the golden test in internal/glfw and, for the
// mirrors that spell out every field with word-sized types, follow from the
// size match on ILP32 too. XkbStateNotifyEvent is the exception: it is a view
// into the XEvent union with a collapsed opaque tail, so only the offsets
// actually read are checked (its total size is LP64-specific).

package xlibcheck_test

import (
	"math/bits"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw/xlibcheck"
)

func TestXlibStructSizes(t *testing.T) {
	checks := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"XEvent", unsafe.Sizeof(glfw.XEvent{}), xlibcheck.SizeofXEvent},
		{"XAnyEvent", unsafe.Sizeof(glfw.XAnyEvent{}), xlibcheck.SizeofXAnyEvent},
		{"XKeyEvent", unsafe.Sizeof(glfw.XKeyEvent{}), xlibcheck.SizeofXKeyEvent},
		{"XButtonEvent", unsafe.Sizeof(glfw.XButtonEvent{}), xlibcheck.SizeofXButtonEvent},
		{"XMotionEvent", unsafe.Sizeof(glfw.XMotionEvent{}), xlibcheck.SizeofXMotionEvent},
		{"XCrossingEvent", unsafe.Sizeof(glfw.XCrossingEvent{}), xlibcheck.SizeofXCrossingEvent},
		{"XFocusChangeEvent", unsafe.Sizeof(glfw.XFocusChangeEvent{}), xlibcheck.SizeofXFocusChangeEvent},
		{"XExposeEvent", unsafe.Sizeof(glfw.XExposeEvent{}), xlibcheck.SizeofXExposeEvent},
		{"XVisibilityEvent", unsafe.Sizeof(glfw.XVisibilityEvent{}), xlibcheck.SizeofXVisibilityEvent},
		{"XDestroyWindowEvent", unsafe.Sizeof(glfw.XDestroyWindowEvent{}), xlibcheck.SizeofXDestroyWindowEvent},
		{"XUnmapEvent", unsafe.Sizeof(glfw.XUnmapEvent{}), xlibcheck.SizeofXUnmapEvent},
		{"XMapEvent", unsafe.Sizeof(glfw.XMapEvent{}), xlibcheck.SizeofXMapEvent},
		{"XReparentEvent", unsafe.Sizeof(glfw.XReparentEvent{}), xlibcheck.SizeofXReparentEvent},
		{"XConfigureEvent", unsafe.Sizeof(glfw.XConfigureEvent{}), xlibcheck.SizeofXConfigureEvent},
		{"XPropertyEvent", unsafe.Sizeof(glfw.XPropertyEvent{}), xlibcheck.SizeofXPropertyEvent},
		{"XSelectionRequestEvent", unsafe.Sizeof(glfw.XSelectionRequestEvent{}), xlibcheck.SizeofXSelectionRequestEvent},
		{"XSelectionEvent", unsafe.Sizeof(glfw.XSelectionEvent{}), xlibcheck.SizeofXSelectionEvent},
		{"XSelectionClearEvent", unsafe.Sizeof(glfw.XSelectionClearEvent{}), xlibcheck.SizeofXSelectionClearEvent},
		{"XClientMessageEvent", unsafe.Sizeof(glfw.XClientMessageEvent{}), xlibcheck.SizeofXClientMessageEvent},
		{"XGenericEventCookie", unsafe.Sizeof(glfw.XGenericEventCookie{}), xlibcheck.SizeofXGenericEventCookie},
		{"XErrorEvent", unsafe.Sizeof(glfw.XErrorEvent{}), xlibcheck.SizeofXErrorEvent},
		{"XSetWindowAttributes", unsafe.Sizeof(glfw.XSetWindowAttributes{}), xlibcheck.SizeofXSetWindowAttributes},
		{"XWindowAttributes", unsafe.Sizeof(glfw.XWindowAttributes{}), xlibcheck.SizeofXWindowAttributes},
		{"Visual", unsafe.Sizeof(glfw.Visual{}), xlibcheck.SizeofVisual},
		{"XVisualInfo", unsafe.Sizeof(glfw.XVisualInfo{}), xlibcheck.SizeofXVisualInfo},
		{"XSizeHints", unsafe.Sizeof(glfw.XSizeHints{}), xlibcheck.SizeofXSizeHints},
		{"XWMHints", unsafe.Sizeof(glfw.XWMHints{}), xlibcheck.SizeofXWMHints},
		{"XClassHint", unsafe.Sizeof(glfw.XClassHint{}), xlibcheck.SizeofXClassHint},
		{"XIMStyles", unsafe.Sizeof(glfw.XIMStyles{}), xlibcheck.SizeofXIMStyles},
		{"XRectangle", unsafe.Sizeof(glfw.XRectangle{}), xlibcheck.SizeofXRectangle},
		{"XkbDescRec", unsafe.Sizeof(glfw.XkbDescRec{}), xlibcheck.SizeofXkbDescRec},
		{"XkbNamesRec", unsafe.Sizeof(glfw.XkbNamesRec{}), xlibcheck.SizeofXkbNamesRec},
		{"XkbKeyNameRec", unsafe.Sizeof(glfw.XkbKeyNameRec{}), xlibcheck.SizeofXkbKeyNameRec},
		{"XkbKeyAliasRec", unsafe.Sizeof(glfw.XkbKeyAliasRec{}), xlibcheck.SizeofXkbKeyAliasRec},
		{"XkbStateRec", unsafe.Sizeof(glfw.XkbStateRec{}), xlibcheck.SizeofXkbStateRec},
		{"XkbAnyEvent", unsafe.Sizeof(glfw.XkbAnyEvent{}), xlibcheck.SizeofXkbAnyEvent},
		{"XRenderDirectFormat", unsafe.Sizeof(glfw.XRenderDirectFormat{}), xlibcheck.SizeofXRenderDirectFormat},
		{"XRenderPictFormat", unsafe.Sizeof(glfw.XRenderPictFormat{}), xlibcheck.SizeofXRenderPictFormat},
		{"XRRModeInfo", unsafe.Sizeof(glfw.XRRModeInfo{}), xlibcheck.SizeofXRRModeInfo},
		{"XRRScreenResources", unsafe.Sizeof(glfw.XRRScreenResources{}), xlibcheck.SizeofXRRScreenResources},
		{"XRROutputInfo", unsafe.Sizeof(glfw.XRROutputInfo{}), xlibcheck.SizeofXRROutputInfo},
		{"XRRCrtcInfo", unsafe.Sizeof(glfw.XRRCrtcInfo{}), xlibcheck.SizeofXRRCrtcInfo},
		{"XineramaScreenInfo", unsafe.Sizeof(glfw.XineramaScreenInfo{}), xlibcheck.SizeofXineramaScreenInfo},
		{"XcursorImage", unsafe.Sizeof(glfw.XcursorImage{}), xlibcheck.SizeofXcursorImage},
		{"XIEventMask", unsafe.Sizeof(glfw.XIEventMask{}), xlibcheck.SizeofXIEventMask},
		{"XIValuatorState", unsafe.Sizeof(glfw.XIValuatorState{}), xlibcheck.SizeofXIValuatorState},
		{"XIRawEvent", unsafe.Sizeof(glfw.XIRawEvent{}), xlibcheck.SizeofXIRawEvent},
	}

	// XkbStateNotifyEvent is a view into the XEvent union; its Go mirror
	// collapses the unread tail into a fixed pad whose end alignment differs
	// between data models, so its total size is only meaningful on LP64.
	if bits.UintSize == 64 {
		checks = append(checks, struct {
			name string
			got  uintptr
			want uintptr
		}{"XkbStateNotifyEvent", unsafe.Sizeof(glfw.XkbStateNotifyEvent{}), xlibcheck.SizeofXkbStateNotifyEvent})
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
		{"XkbType", unsafe.Offsetof(glfw.XkbStateNotifyEvent{}.XkbType), xlibcheck.OffsetofXkbStateNotifyEventXkbType},
		{"Changed", unsafe.Offsetof(glfw.XkbStateNotifyEvent{}.Changed), xlibcheck.OffsetofXkbStateNotifyEventChanged},
		{"Group", unsafe.Offsetof(glfw.XkbStateNotifyEvent{}.Group), xlibcheck.OffsetofXkbStateNotifyEventGroup},
	} {
		if c.got != c.want {
			t.Errorf("offsetof XkbStateNotifyEvent.%s: Go %d, C %d", c.name, c.got, c.want)
		}
	}
}

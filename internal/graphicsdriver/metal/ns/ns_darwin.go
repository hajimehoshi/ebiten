// Copyright 2018 The Ebiten Authors
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

// Package ns provides access to Apple's AppKit API (https://developer.apple.com/documentation/appkit).
//
// This package is in very early stages of development.
// It's a minimal implementation with scope limited to
// supporting the movingtriangle example.
package ns

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
)

// #cgo !ios CFLAGS: -mmacosx-version-min=10.12
//
// #include "ns_darwin.h"
import "C"

// Window is a window that an app displays on the screen.
//
// Reference: https://developer.apple.com/documentation/appkit/nswindow.
type Window struct {
	window uintptr
}

// NewWindow returns a Window that wraps an existing NSWindow * pointer.
func NewWindow(window uintptr) Window {
	return Window{window}
}

// ContentView returns the window's content view, the highest accessible View
// in the window's view hierarchy.
//
// Reference: https://developer.apple.com/documentation/appkit/nswindow/1419160-contentview.
func (w Window) ContentView() View {
	return View{C.Window_ContentView(C.uintptr_t(w.window))}
}

// View is the infrastructure for drawing, printing, and handling events in an app.
//
// Reference: https://developer.apple.com/documentation/appkit/nsview.
type View struct {
	view unsafe.Pointer
}

// SetLayer sets v.layer to l.
//
// Reference: https://developer.apple.com/documentation/appkit/nsview/1483298-layer.
func (v View) SetLayer(l ca.Layer) {
	C.View_SetLayer(v.view, l.Layer())
}

// SetWantsLayer sets v.wantsLayer to wantsLayer.
//
// Reference: https://developer.apple.com/documentation/appkit/nsview/1483695-wantslayer.
func (v View) SetWantsLayer(wantsLayer bool) {
	if wantsLayer {
		C.View_SetWantsLayer(v.view, 1)
	} else {
		C.View_SetWantsLayer(v.view, 0)
	}
}

// IsInFullScreenMode returns a boolean value indicating whether the view is in full screen mode.
//
// Reference: https://developer.apple.com/documentation/appkit/nsview/1483337-infullscreenmode.
func (v View) IsInFullScreenMode() bool {
	return C.View_IsInFullScreenMode(v.view) != 0
}

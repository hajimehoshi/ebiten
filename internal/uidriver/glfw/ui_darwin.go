// Copyright 2016 Hajime Hoshi
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

// +build !js
// +build !ios

package glfw

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// static void currentMonitorPos(uintptr_t windowPtr, int* x, int* y) {
//   NSScreen* screen = [NSScreen mainScreen];
//   if (windowPtr) {
//     NSWindow* window = (NSWindow*)windowPtr;
//     if ([window isVisible]) {
//       // When the window is visible, the window is already initialized.
//       // [NSScreen mainScreen] sometimes tells a lie when the window is put across monitors (#703).
//       screen = [window screen];
//     }
//   }
//   NSDictionary* screenDictionary = [screen deviceDescription];
//   NSNumber* screenID = [screenDictionary objectForKey:@"NSScreenNumber"];
//   CGDirectDisplayID aID = [screenID unsignedIntValue];
//   const CGRect bounds = CGDisplayBounds(aID);
//   *x = bounds.origin.x;
//   *y = bounds.origin.y;
// }
//
// static bool isNativeFullscreen() {
//   return [[NSApplication sharedApplication] currentSystemPresentationOptions] &
//       NSApplicationPresentationFullScreen;
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/internal/glfw"
)

func (u *UserInterface) fromGLFWMonitorPixel(x float64) float64 {
	return x
}

func (u *UserInterface) fromGLFWPixel(x float64) float64 {
	return x
}

func (u *UserInterface) toGLFWPixel(x float64) float64 {
	return x
}

func (u *UserInterface) toFramebufferPixel(x float64) float64 {
	return x
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func currentMonitorByOS(w *glfw.Window) *glfw.Monitor {
	x := C.int(0)
	y := C.int(0)
	// Note: [NSApp mainWindow] is nil when it doesn't have its border. Use w here.
	win := w.GetCocoaWindow()
	C.currentMonitorPos(C.uintptr_t(win), &x, &y)
	for _, m := range glfw.GetMonitors() {
		mx, my := m.GetPos()
		if int(x) == mx && int(y) == my {
			return m
		}
	}
	return nil
}

func (u *UserInterface) nativeWindow() uintptr {
	return u.window.GetCocoaWindow()
}

func (u *UserInterface) isNativeFullscreen() bool {
	return bool(C.isNativeFullscreen())
}

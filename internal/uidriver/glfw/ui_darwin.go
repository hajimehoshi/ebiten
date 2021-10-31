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

//go:build !ios
// +build !ios

package glfw

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// static void currentMonitorPos(uintptr_t windowPtr, int* x, int* y) {
//   @autoreleasepool {
//     NSScreen* screen = [NSScreen mainScreen];
//     if (windowPtr) {
//       NSWindow* window = (NSWindow*)windowPtr;
//       if ([window isVisible]) {
//         // When the window is visible, the window is already initialized.
//         // [NSScreen mainScreen] sometimes tells a lie when the window is put across monitors (#703).
//         screen = [window screen];
//       }
//     }
//     NSDictionary* screenDictionary = [screen deviceDescription];
//     NSNumber* screenID = [screenDictionary objectForKey:@"NSScreenNumber"];
//     CGDirectDisplayID aID = [screenID unsignedIntValue];
//     const CGRect bounds = CGDisplayBounds(aID);
//     *x = bounds.origin.x;
//     *y = bounds.origin.y;
//   }
// }
//
// static bool isNativeFullscreen(uintptr_t windowPtr) {
//   if (!windowPtr) {
//     return false;
//   }
//   NSWindow* window = (NSWindow*)windowPtr;
//   return (window.styleMask & NSWindowStyleMaskFullScreen) != 0;
// }
//
// static void setNativeFullscreen(uintptr_t windowPtr, bool fullscreen) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   if (((window.styleMask & NSWindowStyleMaskFullScreen) != 0) == fullscreen) {
//     return;
//   }
//   bool origResizable = window.styleMask & NSWindowStyleMaskResizable;
//   if (!origResizable) {
//     window.styleMask |= NSWindowStyleMaskResizable;
//   }
//   [window toggleFullScreen:nil];
//   if (!origResizable) {
//     window.styleMask &= ~NSWindowStyleMaskResizable;
//   }
// }
//
// static void adjustViewSize(uintptr_t windowPtr) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   if ((window.styleMask & NSWindowStyleMaskFullScreen) == 0) {
//     return;
//   }
//
//   // Reduce the view height (#1745).
//   // https://stackoverflow.com/questions/27758027/sprite-kit-serious-fps-issue-in-full-screen-mode-on-os-x
//   CGSize windowSize = [window frame].size;
//   NSView* view = [window contentView];
//   CGSize viewSize = [view frame].size;
//   if (windowSize.width != viewSize.width || windowSize.height != viewSize.height) {
//     return;
//   }
//   viewSize.width--;
//   [view setFrameSize:viewSize];
//
//   // NSColor.blackColor (0, 0, 0, 1) didn't work.
//   // Use the transparent color instead.
//   [window setBackgroundColor: [NSColor colorWithSRGBRed:0 green:0 blue:0 alpha:0]];
// }
//
// static void setNativeCursor(int cursorID) {
//   id cursor = [[NSCursor class] performSelector:@selector(arrowCursor)];
//   switch (cursorID) {
//   case 0:
//     cursor = [[NSCursor class] performSelector:@selector(arrowCursor)];
//     break;
//   case 1:
//     cursor = [[NSCursor class] performSelector:@selector(IBeamCursor)];
//     break;
//   case 2:
//     cursor = [[NSCursor class] performSelector:@selector(crosshairCursor)];
//     break;
//   case 3:
//     cursor = [[NSCursor class] performSelector:@selector(pointingHandCursor)];
//     break;
//   case 4:
//     cursor = [[NSCursor class] performSelector:@selector(_windowResizeEastWestCursor)];
//     break;
//   case 5:
//     cursor = [[NSCursor class] performSelector:@selector(_windowResizeNorthSouthCursor)];
//     break;
//   }
//   [cursor push];
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

// videoModeScale must be called from the main thread.
func videoModeScale(m *glfw.Monitor) float64 {
	return 1
}

// fromGLFWMonitorPixel must be called from the main thread.
func (u *UserInterface) fromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	// videoModeScale is always 1 on macOS.
	return x
}

// fromGLFWPixel must be called from the main thread.
func (u *UserInterface) fromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	// NOTE: On macOS, GLFW exposes the device independent coordinate system.
	// Thus, the conversion functions are unnecessary,
	// however we still need the deviceScaleFactor internally
	// so we can create and maintain a HiDPI frame buffer.
	return x
}

// toGLFWPixel must be called from the main thread.
func (u *UserInterface) toGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func initialMonitorByOS() *glfw.Monitor {
	return nil
}

func currentMonitorByOS(w *glfw.Window) *glfw.Monitor {
	x := C.int(0)
	y := C.int(0)
	// Note: [NSApp mainWindow] is nil when it doesn't have its border. Use w here.
	win := w.GetCocoaWindow()
	C.currentMonitorPos(C.uintptr_t(win), &x, &y)
	for _, m := range ensureMonitors() {
		if int(x) == m.x && int(y) == m.y {
			return m.m
		}
	}
	return nil
}

func (u *UserInterface) nativeWindow() uintptr {
	return u.window.GetCocoaWindow()
}

func (u *UserInterface) isNativeFullscreen() bool {
	return bool(C.isNativeFullscreen(C.uintptr_t(u.window.GetCocoaWindow())))
}

func (u *UserInterface) setNativeCursor(shape driver.CursorShape) {
	C.setNativeCursor(C.int(shape))
}

func (u *UserInterface) isNativeFullscreenAvailable() bool {
	// TODO: If the window is transparent, we should use GLFW's windowed fullscreen (#1822, #1857).
	// However, if the user clicks the green button, should this window be in native fullscreen mode?
	return true
}

func (u *UserInterface) setNativeFullscreen(fullscreen bool) {
	// Toggling fullscreen might ignore events like keyUp. Ensure that events are fired.
	glfw.WaitEventsTimeout(0.1)
	C.setNativeFullscreen(C.uintptr_t(u.window.GetCocoaWindow()), C.bool(fullscreen))
}

func (u *UserInterface) adjustViewSize() {
	if u.Graphics().IsGL() {
		return
	}
	C.adjustViewSize(C.uintptr_t(u.window.GetCocoaWindow()))
}

func initializeWindowAfterCreation(w *glfw.Window) {
}

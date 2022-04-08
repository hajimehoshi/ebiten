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

//go:build !ios && !ebitencbackend
// +build !ios,!ebitencbackend

package ui

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// @interface EbitenWindowDelegate : NSObject <NSWindowDelegate>
// // origPos is the window's original position. This is valid only when the application is in the fullscreen mode.
// @property CGPoint origPos;
// @end
//
// @implementation EbitenWindowDelegate {
//   id<NSWindowDelegate> origDelegate_;
//   bool origResizable_;
// }
//
// - (instancetype)initWithOrigDelegate:(id<NSWindowDelegate>)origDelegate {
//   self = [super init];
//   if (self != nil) {
//     origDelegate_ = origDelegate;
//   }
//   return self;
// }
//
// // The method set of origDelegate_ must sync with GLFWWindowDelegate's implementation.
// // See cocoa_window.m in GLFW.
// - (BOOL)windowShouldClose:(id)sender {
//   return [origDelegate_ windowShouldClose:sender];
// }
// - (void)windowDidResize:(NSNotification *)notification {
//   [origDelegate_ windowDidResize:notification];
// }
// - (void)windowDidMove:(NSNotification *)notification {
//   [origDelegate_ windowDidMove:notification];
// }
// - (void)windowDidMiniaturize:(NSNotification *)notification {
//   [origDelegate_ windowDidMiniaturize:notification];
// }
// - (void)windowDidDeminiaturize:(NSNotification *)notification {
//   [origDelegate_ windowDidDeminiaturize:notification];
// }
// - (void)windowDidBecomeKey:(NSNotification *)notification {
//   [origDelegate_ windowDidBecomeKey:notification];
// }
// - (void)windowDidResignKey:(NSNotification *)notification {
//   [origDelegate_ windowDidResignKey:notification];
// }
// - (void)windowDidChangeOcclusionState:(NSNotification* )notification {
//   [origDelegate_ windowDidChangeOcclusionState:notification];
// }
//
// - (void)pushResizableState:(NSWindow*)window {
//   origResizable_ = window.styleMask & NSWindowStyleMaskResizable;
//   if (!origResizable_) {
//     window.styleMask |= NSWindowStyleMaskResizable;
//   }
// }
//
// - (void)popResizableState:(NSWindow*)window {
//   if (!origResizable_) {
//     window.styleMask &= ~NSWindowStyleMaskResizable;
//   }
//   origResizable_ = false;
// }
//
// - (void)windowWillEnterFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self pushResizableState:window];
//   self->_origPos = [window frame].origin;
// }
//
// - (void)windowDidEnterFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self popResizableState:window];
// }
//
// - (void)windowWillExitFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self pushResizableState:window];
// }
//
// - (void)windowDidExitFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self popResizableState:window];
//   [window setFrameOrigin:self->_origPos];
// }
//
// @end
//
// static void initializeWindow(uintptr_t windowPtr) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   // This delegate is never released. This assumes that the window lives until the process lives.
//   window.delegate = [[EbitenWindowDelegate alloc] initWithOrigDelegate:window.delegate];
// }
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
//
//   // Even though EbitenWindowDelegate is used, this hack is still required.
//   // toggleFullscreen doesn't work when the window is not resizable.
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
//   // Apparently, adjusting the view size is not needed as of macOS 12 (#1745).
//   static int majorVersion = 0;
//   if (majorVersion == 0) {
//     majorVersion = [[NSProcessInfo processInfo] operatingSystemVersion].majorVersion;
//   }
//   if (majorVersion >= 12) {
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
// static void windowOriginalPosition(uintptr_t windowPtr, int* x, int* y) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   CGPoint pos;
//   EbitenWindowDelegate* delegate = (EbitenWindowDelegate*)window.delegate;
//   pos = delegate.origPos;
//   *x = pos.x;
//   *y = pos.y;
// }
//
// static void setWindowOriginalPosition(uintptr_t windowPtr, int x, int y) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   EbitenWindowDelegate* delegate = (EbitenWindowDelegate*)window.delegate;
//   CGPoint pos;
//   pos.x = x;
//   pos.y = y;
//   delegate.origPos = pos;
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
//
// static void currentMouseLocation(int* x, int* y) {
//   NSPoint location = [NSEvent mouseLocation];
//   *x = (int)(location.x);
//   *y = (int)(location.y);
// }
//
// static void setAllowFullscreen(uintptr_t windowPtr, bool allowFullscreen) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   if (allowFullscreen) {
//     window.collectionBehavior |= NSWindowCollectionBehaviorFullScreenPrimary;
//   } else {
//     window.collectionBehavior &= ~NSWindowCollectionBehaviorFullScreenPrimary;
//   }
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverGetterImpl struct {
	transparent bool
}

func (g *graphicsDriverGetterImpl) getAuto() graphicsdriver.Graphics {
	if m := g.getMetal(); m != nil {
		return m
	}
	return g.getOpenGL()
}

func (*graphicsDriverGetterImpl) getOpenGL() graphicsdriver.Graphics {
	if g := opengl.Get(); g != nil {
		return g
	}
	return nil
}

func (*graphicsDriverGetterImpl) getDirectX() graphicsdriver.Graphics {
	return nil
}

func (*graphicsDriverGetterImpl) getMetal() graphicsdriver.Graphics {
	if m := metal.Get(); m != nil {
		return m
	}
	return nil
}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

// dipFromGLFWMonitorPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	return x
}

// dipFromGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	// NOTE: On macOS, GLFW exposes the device independent coordinate system.
	// Thus, the conversion functions are unnecessary,
	// however we still need the deviceScaleFactor internally
	// so we can create and maintain a HiDPI frame buffer.
	return x
}

// dipToGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipToGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x
}

func (u *userInterfaceImpl) adjustWindowPosition(x, y int, monitor *glfw.Monitor) (int, int) {
	return x, y
}

func flipY(y int) int {
	for _, m := range ensureMonitors() {
		if m.x == 0 && m.y == 0 {
			y = -y
			y += m.vm.Height
			break
		}
	}
	return y
}

func initialMonitorByOS() (*glfw.Monitor, error) {
	var cx, cy C.int
	C.currentMouseLocation(&cx, &cy)
	x, y := int(cx), flipY(int(cy))

	// Find the monitor including the cursor.
	for _, m := range ensureMonitors() {
		w, h := m.vm.Width, m.vm.Height
		if x >= m.x && x < m.x+w && y >= m.y && y < m.y+h {
			return m.m, nil
		}
	}

	return nil, nil
}

func monitorFromWindowByOS(w *glfw.Window) *glfw.Monitor {
	var x, y C.int
	C.currentMonitorPos(C.uintptr_t(w.GetCocoaWindow()), &x, &y)
	for _, m := range ensureMonitors() {
		if int(x) == m.x && int(y) == m.y {
			return m.m
		}
	}
	return nil
}

func (u *userInterfaceImpl) nativeWindow() uintptr {
	return u.window.GetCocoaWindow()
}

func (u *userInterfaceImpl) isNativeFullscreen() bool {
	return bool(C.isNativeFullscreen(C.uintptr_t(u.window.GetCocoaWindow())))
}

func (u *userInterfaceImpl) setNativeCursor(shape CursorShape) {
	C.setNativeCursor(C.int(shape))
}

func (u *userInterfaceImpl) isNativeFullscreenAvailable() bool {
	// TODO: If the window is transparent, we should use GLFW's windowed fullscreen (#1822, #1857).
	// However, if the user clicks the green button, should this window be in native fullscreen mode?
	return true
}

func (u *userInterfaceImpl) setNativeFullscreen(fullscreen bool) {
	// Toggling fullscreen might ignore events like keyUp. Ensure that events are fired.
	glfw.WaitEventsTimeout(0.1)
	C.setNativeFullscreen(C.uintptr_t(u.window.GetCocoaWindow()), C.bool(fullscreen))
}

func (u *userInterfaceImpl) adjustViewSize() {
	if u.graphicsDriver.IsGL() {
		return
	}
	C.adjustViewSize(C.uintptr_t(u.window.GetCocoaWindow()))
}

func (u *userInterfaceImpl) setWindowResizingModeForOS(mode WindowResizingMode) {
	allowFullscreen := mode == WindowResizingModeOnlyFullscreenEnabled ||
		mode == WindowResizingModeEnabled
	C.setAllowFullscreen(C.uintptr_t(u.window.GetCocoaWindow()), C.bool(allowFullscreen))
}

func initializeWindowAfterCreation(w *glfw.Window) {
	// TODO: Register NSWindowWillEnterFullScreenNotification and so on.
	// Enable resizing temporary before making the window fullscreen.
	C.initializeWindow(C.uintptr_t(w.GetCocoaWindow()))
}

func (u *userInterfaceImpl) origWindowPosByOS() (int, int, bool) {
	if !u.isNativeFullscreen() {
		return invalidPos, invalidPos, true
	}
	var cx, cy C.int
	C.windowOriginalPosition(C.uintptr_t(u.window.GetCocoaWindow()), &cx, &cy)
	x := int(cx)
	y := flipY(int(cy)) - u.windowHeightInDIP
	return x, y, true
}

func (u *userInterfaceImpl) setOrigWindowPosByOS(x, y int) bool {
	if !u.isNativeFullscreen() {
		return true
	}
	cx := C.int(x)
	cy := C.int(flipY(y + u.windowHeightInDIP))
	C.setWindowOriginalPosition(C.uintptr_t(u.window.GetCocoaWindow()), cx, cy)
	return true
}

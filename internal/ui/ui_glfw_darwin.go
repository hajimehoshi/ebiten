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

//go:build !ios && !nintendosdk
// +build !ios,!nintendosdk

package ui

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// @interface EbitenWindowDelegate : NSObject <NSWindowDelegate>
// // origPos is the window's original position. This is valid only when the application is in the fullscreen mode.
// @property CGPoint origPos;
// // origSize is the window's original size.
// @property CGSize origSize;
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
//   self->_origSize = [window frame].size;
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
//   NSUInteger origCollectionBehavior = window.collectionBehavior;
//   bool origFullscreen = origCollectionBehavior & NSWindowCollectionBehaviorFullScreenPrimary;
//   if (!origFullscreen) {
//     NSUInteger collectionBehavior = origCollectionBehavior;
//     collectionBehavior |= NSWindowCollectionBehaviorFullScreenPrimary;
//     collectionBehavior &= ~NSWindowCollectionBehaviorFullScreenNone;
//     window.collectionBehavior = collectionBehavior;
//   }
//   [window toggleFullScreen:nil];
//   if (!origFullscreen) {
//     window.collectionBehavior = origCollectionBehavior;
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
//   EbitenWindowDelegate* delegate = (EbitenWindowDelegate*)window.delegate;
//   CGPoint pos = delegate.origPos;
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
// static void windowOriginalSize(uintptr_t windowPtr, int* width, int* height) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   EbitenWindowDelegate* delegate = (EbitenWindowDelegate*)window.delegate;
//   CGSize size = delegate.origSize;
//   *width = size.width;
//   *height = size.height;
// }
//
// static void setWindowOriginalSize(uintptr_t windowPtr, int width, int height) {
//   NSWindow* window = (NSWindow*)windowPtr;
//   EbitenWindowDelegate* delegate = (EbitenWindowDelegate*)window.delegate;
//   CGSize size;
//   size.width = width;
//   size.height = height;
//   delegate.origSize = size;
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
//   uint collectionBehavior = 0;
//   if (allowFullscreen) {
//     collectionBehavior |= NSWindowCollectionBehaviorManaged;
//     collectionBehavior |= NSWindowCollectionBehaviorFullScreenPrimary;
//   } else {
//     collectionBehavior |= NSWindowCollectionBehaviorFullScreenNone;
//   }
//   window.collectionBehavior = collectionBehavior;
// }
import "C"

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	transparent bool
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	m, err1 := g.newMetal()
	if err1 == nil {
		return m, GraphicsLibraryMetal, nil
	}
	o, err2 := g.newOpenGL()
	if err2 == nil {
		return o, GraphicsLibraryOpenGL, nil
	}
	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: Metal: %v, OpenGL: %v", err1, err2)
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, nil
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return metal.NewGraphics()
}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

type userInterfaceImplNative struct{}

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

func (u *userInterfaceImpl) origWindowPos() (int, int) {
	if !u.isNativeFullscreen() {
		return invalidPos, invalidPos
	}
	var cx, cy C.int
	C.windowOriginalPosition(C.uintptr_t(u.window.GetCocoaWindow()), &cx, &cy)
	x := int(cx)
	_, h := u.origWindowSizeInDIP()
	y := flipY(int(cy)) - h
	return x, y
}

func (u *userInterfaceImpl) setOrigWindowPos(x, y int) {
	if !u.isNativeFullscreen() {
		return
	}
	cx := C.int(x)
	_, h := u.origWindowSizeInDIP()
	cy := C.int(flipY(y + h))
	C.setWindowOriginalPosition(C.uintptr_t(u.window.GetCocoaWindow()), cx, cy)
}

func (u *userInterfaceImpl) origWindowSizeInDIP() (int, int) {
	// TODO: Make these values consistent with the original positions that are updated only when the app is in fullscreen.
	var cw, ch C.int
	C.windowOriginalSize(C.uintptr_t(u.window.GetCocoaWindow()), &cw, &ch)
	w := int(u.dipFromGLFWPixel(float64(cw), u.currentMonitor()))
	h := int(u.dipFromGLFWPixel(float64(ch), u.currentMonitor()))
	return w, h
}

func (u *userInterfaceImpl) setOrigWindowSizeInDIP(width, height int) {
	cw := C.int(u.dipFromGLFWPixel(float64(width), u.currentMonitor()))
	ch := C.int(u.dipFromGLFWPixel(float64(height), u.currentMonitor()))
	C.setWindowOriginalSize(C.uintptr_t(u.window.GetCocoaWindow()), cw, ch)
}

func (u *userInterfaceImplNative) initialize() {
}

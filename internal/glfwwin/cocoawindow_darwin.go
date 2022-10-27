package glfwwin

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

var (
	class_GLFWWindowDelegate objc.Class
	class_GLFWWindow         objc.Class

	sel_initWithGlfwWindow                          = objc.RegisterName("initWithGlfwWindow")
	sel_initWithContentRect_styleMask_backing_defer = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
)

type objcGLFWWindow struct {
	objc.ID
}

func glfwWindow_alloc() objcGLFWWindow {
	return objcGLFWWindow{objc.ID(class_GLFWWindow).Send(sel_alloc)}
}

func (w objcGLFWWindow) InitWithContentRectStyleMaskBackingDefer(contentRect cocoa.NSRect, style cocoa.NSWindowStyleMask, backing cocoa.NSBackingStoreType, flag bool) objc.ID {
	sig := cocoa.NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_GLFWWindow), sel_initWithContentRect_styleMask_backing_defer)
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(w.ID)
	inv.SetSelector(sel_initWithContentRect_styleMask_backing_defer)
	inv.SetArgumentAtIndex(unsafe.Pointer(&contentRect), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&style), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&backing), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&flag), 5)
	inv.Invoke()
	var ret objc.ID
	inv.GetReturnValue(unsafe.Pointer(&ret))
	return ret
}

func init() {
	class_GLFWWindowDelegate = objc.AllocateClassPair(objc.GetClass("NSObject"), "GLFWWindowDelegate", 0)
	class_GLFWWindowDelegate.AddIvar("window", objc.ID(0), "@")
	offset := class_GLFWWindowDelegate.InstanceVariable("window").Offset()
	class_GLFWWindowDelegate.AddMethod(sel_initWithGlfwWindow, objc.NewIMP(func(self objc.ID, cmd objc.SEL, w *Window) objc.ID {
		self = self.SendSuper(sel_init)
		if self != 0 {
			*(**Window)(unsafe.Pointer(uintptr(unsafe.Pointer(self)) + offset)) = w
		}
		return self
	}), "@@:^Window")
	/*

		- (BOOL)windowShouldClose:(id)sender
		{
		    _glfwInputWindowCloseRequest(window);
		    return NO;
		}

		- (void)windowDidResize:(NSNotification *)notification
		{
		    if (window->context.source == GLFW_NATIVE_CONTEXT_API)
		        [window->context.nsgl.object update];

		    if (_glfw.ns.disabledCursorWindow == window)
		        _glfwCenterCursorInContentArea(window);

		    const int maximized = [window->ns.object isZoomed];
		    if (window->ns.maximized != maximized)
		    {
		        window->ns.maximized = maximized;
		        _glfwInputWindowMaximize(window, maximized);
		    }

		    const NSRect contentRect = [window->ns.view frame];
		    const NSRect fbRect = [window->ns.view convertRectToBacking:contentRect];

		    if (fbRect.size.width != window->ns.fbWidth ||
		        fbRect.size.height != window->ns.fbHeight)
		    {
		        window->ns.fbWidth  = fbRect.size.width;
		        window->ns.fbHeight = fbRect.size.height;
		        _glfwInputFramebufferSize(window, fbRect.size.width, fbRect.size.height);
		    }

		    if (contentRect.size.width != window->ns.width ||
		        contentRect.size.height != window->ns.height)
		    {
		        window->ns.width  = contentRect.size.width;
		        window->ns.height = contentRect.size.height;
		        _glfwInputWindowSize(window, contentRect.size.width, contentRect.size.height);
		    }
		}

		- (void)windowDidMove:(NSNotification *)notification
		{
		    if (window->context.source == GLFW_NATIVE_CONTEXT_API)
		        [window->context.nsgl.object update];

		    if (_glfw.ns.disabledCursorWindow == window)
		        _glfwCenterCursorInContentArea(window);

		    int x, y;
		    _glfwGetWindowPosCocoa(window, &x, &y);
		    _glfwInputWindowPos(window, x, y);
		}

		- (void)windowDidMiniaturize:(NSNotification *)notification
		{
		    if (window->monitor)
		        releaseMonitor(window);

		    _glfwInputWindowIconify(window, GLFW_TRUE);
		}

		- (void)windowDidDeminiaturize:(NSNotification *)notification
		{
		    if (window->monitor)
		        acquireMonitor(window);

		    _glfwInputWindowIconify(window, GLFW_FALSE);
		}

		- (void)windowDidBecomeKey:(NSNotification *)notification
		{
		    if (_glfw.ns.disabledCursorWindow == window)
		        _glfwCenterCursorInContentArea(window);

		    _glfwInputWindowFocus(window, GLFW_TRUE);
		    updateCursorMode(window);
		}

		- (void)windowDidResignKey:(NSNotification *)notification
		{
		    if (window->monitor && window->autoIconify)
		        _glfwIconifyWindowCocoa(window);

		    _glfwInputWindowFocus(window, GLFW_FALSE);
		}

		- (void)windowDidChangeOcclusionState:(NSNotification* )notification
		{
		    if ([window->ns.object occlusionState] & NSWindowOcclusionStateVisible)
		        window->ns.occluded = GLFW_FALSE;
		    else
		        window->ns.occluded = GLFW_TRUE;
		}

		@end
	*/
	class_GLFWWindowDelegate.Register()

	class_GLFWWindow = objc.AllocateClassPair(objc.GetClass("NSWindow"), "GLFWWindow", 0)
	class_GLFWWindow.AddMethod(objc.RegisterName("canBecomeKeyWindow"), objc.NewIMP(func(self objc.ID, cmd objc.SEL) int {
		// Required for NSWindowStyleMaskBorderless windows
		return 1 // TODO: support bool callbacks
	}), "B@:")
	class_GLFWWindow.AddMethod(objc.RegisterName("canBecomeMainWindow"), objc.NewIMP(func(self objc.ID, cmd objc.SEL) int {
		// Required for NSWindowStyleMaskBorderless windows
		return 1 // TODO: support bool callbacks
	}), "B@:")
	class_GLFWWindow.Register()
}

func platformGetKeyScancode(key Key) int {
	return _glfw.state.scancodes[key]
}

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	w.state.delegate = objc.ID(class_GLFWWindowDelegate).Send(sel_alloc).Send(sel_initWithGlfwWindow, unsafe.Pointer(w))
	if w.state.delegate == 0 {
		return fmt.Errorf("cocoa: failed to create window delegate")
	}
	var contentRect cocoa.NSRect
	_ = contentRect
	if w.monitor != nil {
		//        GLFWvidmode mode;
		//        int xpos, ypos;
		//
		//        _glfwGetVideoModeCocoa(window->monitor, &mode);
		//        _glfwGetMonitorPosCocoa(window->monitor, &xpos, &ypos);
		//
		// contentRect = cocoa.NSRect{xpos, ypos, mode.width, mode.height},
		panic("todo")
	} else {
		//        if (wndconfig->xpos == GLFW_ANY_POSITION ||
		//            wndconfig->ypos == GLFW_ANY_POSITION)
		{
			contentRect = cocoa.NSRect{Origin: cocoa.CGPoint{}, Size: cocoa.NSSize{Width: cocoa.CGFloat(wndconfig.width), Height: cocoa.CGFloat(wndconfig.height)}}
		}
		//        else
		//        {
		//            const int xpos = wndconfig->xpos;
		//            const int ypos = _glfwTransformYCocoa(wndconfig->ypos + wndconfig->height - 1);
		//            contentRect = NSMakeRect(xpos, ypos, wndconfig->width, wndconfig->height);
		//        }
	}

	styleMask := cocoa.NSWindowStyleMaskMiniaturizable

	if w.monitor != nil || !w.decorated {
		styleMask |= cocoa.NSWindowStyleMaskBorderless
	} else {
		styleMask |= cocoa.NSWindowStyleMaskTitled | cocoa.NSWindowStyleMaskClosable

		if w.resizable {
			styleMask |= cocoa.NSWindowStyleMaskResizable
		}

	}
	w.state.object = glfwWindow_alloc().InitWithContentRectStyleMaskBackingDefer(contentRect, styleMask, cocoa.NSBackingStoreBuffered, false)

	if w.state.object == 0 {
		return fmt.Errorf("cocoa: failed to create window")
	}

	//    if (window->monitor)
	//        [window->ns.object setLevel:NSMainMenuWindowLevel + 1];
	//    else
	//    {
	//        if (wndconfig->xpos == GLFW_ANY_POSITION ||
	//            wndconfig->ypos == GLFW_ANY_POSITION)
	//        {
	//            [(NSWindow*) window->ns.object center];
	//            _glfw.ns.cascadePoint =
	//                NSPointToCGPoint([window->ns.object cascadeTopLeftFromPoint:
	//                                NSPointFromCGPoint(_glfw.ns.cascadePoint)]);
	//        }
	//
	//        if (wndconfig->resizable)
	//        {
	//            const NSWindowCollectionBehavior behavior =
	//                NSWindowCollectionBehaviorFullScreenPrimary |
	//                NSWindowCollectionBehaviorManaged;
	//            [window->ns.object setCollectionBehavior:behavior];
	//        }
	//        else
	//        {
	//            const NSWindowCollectionBehavior behavior =
	//                NSWindowCollectionBehaviorFullScreenNone;
	//            [window->ns.object setCollectionBehavior:behavior];
	//        }
	//
	//        if (wndconfig->floating)
	//            [window->ns.object setLevel:NSFloatingWindowLevel];
	//
	//        if (wndconfig->maximized)
	//            [window->ns.object zoom:nil];
	//    }
	//
	//    if (strlen(wndconfig->ns.frameName))
	//        [window->ns.object setFrameAutosaveName:@(wndconfig->ns.frameName)];
	//
	//    window->ns.view = [[GLFWContentView alloc] initWithGlfwWindow:window];
	//    window->ns.retina = wndconfig->ns.retina;
	//
	//    if (fbconfig->transparent)
	//    {
	//        [window->ns.object setOpaque:NO];
	//        [window->ns.object setHasShadow:NO];
	//        [window->ns.object setBackgroundColor:[NSColor clearColor]];
	//    }
	//
	//    [window->ns.object setContentView:window->ns.view];
	//    [window->ns.object makeFirstResponder:window->ns.view];
	//    [window->ns.object setTitle:@(wndconfig->title)];
	//    [window->ns.object setDelegate:window->ns.delegate];
	//    [window->ns.object setAcceptsMouseMovedEvents:YES];
	//    [window->ns.object setRestorable:NO];
	//
	//#if MAC_OS_X_VERSION_MAX_ALLOWED >= 101200
	//    if ([window->ns.object respondsToSelector:@selector(setTabbingMode:)])
	//        [window->ns.object setTabbingMode:NSWindowTabbingModeDisallowed];
	//#endif
	//
	//    _glfwGetWindowSizeCocoa(window, &window->ns.width, &window->ns.height);
	//    _glfwGetFramebufferSizeCocoa(window, &window->ns.fbWidth, &window->ns.fbHeight);
	//
	return nil
}

func (w *Window) platformDestroyWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	//    if (_glfw.ns.disabledCursorWindow == window)
	//        _glfw.ns.disabledCursorWindow = NULL;
	//
	//    [window->ns.object orderOut:nil];
	//
	//    if (window->monitor)
	//        releaseMonitor(window);
	//
	//    if (window->context.destroy)
	//        window->context.destroy(window);
	//
	win := cocoa.NSWindow{ID: w.state.object}
	win.SetDelegate(0)
	//    [window->ns.delegate release];
	w.state.delegate = 0

	view := cocoa.NSView{ID: w.state.view}
	_ = view
	//    [window->ns.view release];
	w.state.view = 0

	//    [window->ns.object close];
	//    window->ns.object = nil;
	//
	// HACK: Allow Cocoa to catch up before returning
	return platformPollEvents()
}

func (w *Window) platformSetWindowTitle(title string) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	t := cocoa.NSString_alloc().InitWithUTF8String(title)
	win := cocoa.NSWindow{w.state.object}
	win.SetTitle(t)
	// HACK: Set the miniwindow title explicitly as setTitle: doesn't update it
	//       if the window lacks NSWindowStyleMaskTitled
	win.SetMiniwindowTitle(t)
	return nil
}

func (w *Window) updateWindowStyles() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformIconifyWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformRestoreWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformMaximizeWindow() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformShowWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	cocoa.NSWindow{ID: w.state.object}.OrderFront(0)
}

func (w *Window) platformHideWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowIconified() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	return cocoa.NSWindow{ID: w.state.object}.IsMiniaturized()
}

func (w *Window) platformWindowVisible() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowMaximized() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowHovered() (bool, error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformFramebufferTransparent() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func platformPollEvents() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	distantPast := cocoa.NSDate_distantPast()
	for {
		event := cocoa.NSApp.NextEventMatchingMaskUntilDateInModeDequeue(
			cocoa.NSEventMaskAny,
			distantPast,
			cocoa.NSDefaultRunLoopMode,
			true,
		)
		if event.ID == 0 {
			break
		}
		cocoa.NSApp.SendEvent(event)
	}
	return nil
}

func platformWaitEvents() error {
	panic("NOT IMPLEMENTED")
}

func platformWaitEventsTimeout(timeout float64) error {
	panic("NOT IMPLEMENTED")
}

func platformPostEmptyEvent() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformRequestWindowAttention() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformFocusWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	// Make us the active application
	// HACK: This is here to prevent applications using only hidden windows from
	//       being activated, but should probably not be done every time any
	//       window is shown
	cocoa.NSApp.ActivateIgnoringOtherApps(true)
	cocoa.NSWindow{ID: w.state.object}.MakeKeyAndOrderFront(0)
	return nil
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	contentRect := cocoa.NSView{ID: w.state.view}.Frame()
	return int(contentRect.Size.Width), int(contentRect.Size.Height), nil
}

func (w *Window) platformSetCursorPos(f float64, f2 float64) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowSize(width, height int) error {
	log.Println("glfw: platformSetWindowSize: NOT IMPLEMENTED")
	return nil
}

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	log.Println("glfw: platformGetCursorPos: NOT IMPLEMENTED")
	return
}

func (w *Window) platformSetCursorMode(mode int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	//    if (cursorInContentArea(window))
	//        updateCursorImage(window);
	return nil
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	log.Println("glfw: platformSetWindowPos: NOT IMPLEMENTED")
	return nil
}

func platformRawMouseMotionSupported() bool {
	panic("NOT IMPLEMENTED")
	return true
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowFocused() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	return cocoa.NSWindow{w.state.object}.IsKeyWindow()
	// @autoreleasepool {
	//    return [window->ns.object isKeyWindow];
	//    } // autoreleasepool
}

func (c *Cursor) platformCreateCursor(image *Image, xhot, yhot int) error {
	panic("NOT IMPLEMENTED")
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	var cursorSelector objc.SEL
	// HACK: Try to use a private message
	switch shape {
	case HResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeEastWestCursor")
	case VResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthSouthCursor")
	case NWSEResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthWestSouthEastCursor")
	case NESWResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthEastSouthWestCursor")
	}
	if cursorSelector != 0 && cocoa.NSCursor_respondsToSelector(cursorSelector) {
		object := cocoa.NSCursor_performSelector(cursorSelector)
		// TODO: check kind
		//if ([object isKindOfClass:[NSCursor class]]) {
		c.state.object = object
		//}
	}
	if c.state.object == 0 {
		switch shape {
		case ArrowCursor:
			//cursor->ns.object = [NSCursor arrowCursor];
			panic("TODO")
		case IBeamCursor:
			c.state.object = cocoa.NSCursor_IBeamCursor().ID
		case CrosshairCursor:
			c.state.object = cocoa.NSCursor_crosshairCursor().ID
		case HandCursor:
			c.state.object = cocoa.NSCursor_pointingHandCursor().ID
		case HResizeCursor:
			//cursor->ns.object = [NSCursor resizeLeftRightCursor];
			panic("TODO")
		case VResizeCursor:
			// cursor->ns.object = [NSCursor resizeUpDownCursor];
			panic("TODO")
		case AllResizeCursor:
			// cursor->ns.object = [NSCursor closedHandCursor];
			panic("TODO")
		case NotAllowedCursor:
			//cursor->ns.object = [NSCursor operationNotAllowedCursor];
			panic("TODO")
		}
	}

	if c.state.object == 0 {
		return fmt.Errorf("cocoa: standard cursor shape unavailable")
	}

	cocoa.NSObject_retain(c.state.object)
	return nil
}

func (c *Cursor) platformDestroyCursor() error {
	log.Println("glfw: platformDestroyCursor: NOT IMPLEMENTED")
	return nil
}

func platformSetClipboardString(str string) error {
	panic("glfwwin: platformSetClipboardString is not implemented")
}

func platformGetClipboardString() (string, error) {
	panic("glfwwin: platformGetClipboardString is not implemented")
}

func (w *Window) GetCocoaWindow() (uintptr, error) {
	//  _GLFWwindow* window = (_GLFWwindow*) handle;
	//    _GLFW_REQUIRE_INIT_OR_RETURN(nil);
	//
	//    if (_glfw.platform.platformID != GLFW_PLATFORM_COCOA)
	//    {
	//        _glfwInputError(GLFW_PLATFORM_UNAVAILABLE,
	//                        "Cocoa: Platform not initialized");
	//        return NULL;
	//    }
	//
	return uintptr(w.state.object), nil
}
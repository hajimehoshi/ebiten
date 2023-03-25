// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfwwin

// #cgo LDFLAGS: -framework CoreGraphics
// #include <CoreGraphics/CoreGraphics.h>
import "C"
import "C"
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
	class_GLFWContentView    objc.Class

	sel_initWithGlfwWindow = objc.RegisterName("initWithGlfwWindow:")
)

type windowDelegate struct {
	isa    objc.Class `objc:"GLFWWindowDelegate : NSObject"`
	window *Window
}

func (d *windowDelegate) InitWithGlfwWindow(cmd objc.SEL, w *Window) objc.ID {
	self := (*objc.ID)(unsafe.Pointer(&d)).SendSuper(sel_init)
	if self != 0 {
		(*(**windowDelegate)(unsafe.Pointer(&self))).window = w
	}
	return self
}

func (d *windowDelegate) WindowShouldClose(cmd objc.SEL, sender objc.ID) bool {
	d.window.inputWindowCloseRequest()
	return false
}

/*
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
*/

func (d *windowDelegate) WindowDidMove(cmd objc.SEL, _ objc.ID /* *NSNotification */) {
	if d.window.context.source == NativeContextAPI {
		//[window->context.nsgl.object update];
		panic("TODO")
	}
	//    if (_glfw.ns.disabledCursorWindow == window)
	//        _glfwCenterCursorInContentArea(window);
	//
	var x, y, _ = d.window.GetPos()
	d.window.inputCursorPos(float64(x), float64(y))
}

/*- (void)windowDidMiniaturize:(NSNotification *)notification
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
}*/

func (d *windowDelegate) WindowDidBecomeKey(cmd objc.SEL, _ objc.ID) {
	if _glfw.state.disableCursorWindow == d.window {
		err := d.window.centerCursorInContentArea()
		if err != nil {
			panic(err)
		}
	}
	d.window.inputWindowFocus(true)
	fmt.Println("WindowDidBecomeKey: FIXME")
	/*	updateCursorMode(window);*/
}

func (d *windowDelegate) WindowDidResignKey(cmd objc.SEL, _ objc.ID) {
	if d.window.monitor != nil && d.window.autoIconify {
		d.window.platformIconifyWindow()
	}
	d.window.inputWindowFocus(false)
}

func (d *windowDelegate) WindowDidChangeOcclusionState(cmd objc.SEL, _ objc.ID) {
	fmt.Println("WindowDidChangeOcclusionState: TODO")
	//    if ([window->ns.object occlusionState] & NSWindowOcclusionStateVisible)
	//        window->ns.occluded = GLFW_FALSE;
	//    else
	//        window->ns.occluded = GLFW_TRUE;
}

func (d *windowDelegate) Selector(s string) objc.SEL {
	switch s {
	case "WindowDidBecomeKey":
		return objc.RegisterName("windowDidBecomeKey:")
	case "WindowDidMove":
		return objc.RegisterName("windowDidMove:")
	case "InitWithGlfwWindow":
		return sel_initWithGlfwWindow
	case "WindowDidChangeOcclusionState":
		return objc.RegisterName("windowDidChangeOcclusionState:")
	case "WindowDidResignKey":
		return objc.RegisterName("windowDidResignKey:")
	case "WindowShouldClose":
		return objc.RegisterName("windowShouldClose:")
	default:
		return 0
	}
}

type window struct {
	isa objc.Class `objc:"GLFWWindow : NSWindow"`
	_   [420]byte
}

func (w *window) CanBecomeKeyWindow(cmd objc.SEL) bool {
	// Required for NSWindowStyleMaskBorderless windows
	return true
}

func (w *window) CanBecomeMainWindow(cmd objc.SEL) bool {
	// Required for NSWindowStyleMaskBorderless windows
	return true
}

func (w *window) Selector(s string) objc.SEL {
	switch s {
	case "CanBecomeKeyWindow":
		return objc.RegisterName("canBecomeKeyWindow")
	case "CanBecomeMainWindow":
		return objc.RegisterName("canBecomeMainWindow")
	default:
		return 0
	}
}

type contentView struct {
	isa    objc.Class `objc:"GLFWContentView : NSView <NSTextInputClient>"`
	_      [552]byte
	window *Window
	//    NSTrackingArea* trackingArea;
	//    NSMutableAttributedString* markedText;
}

func (v *contentView) InitWithGlfwWindow(cmd objc.SEL, w *Window) objc.ID {
	fmt.Println("FIXME: InitWithGlfwWindow")
	self := (*objc.ID)(unsafe.Pointer(&v)).SendSuper(sel_init)
	if self != 0 {
		(*(**contentView)(unsafe.Pointer(&self))).window = w
		//        trackingArea = nil;
		//        markedText = [[NSMutableAttributedString alloc] init];
		//
		//        [self updateTrackingAreas];
		//        [self registerForDraggedTypes:@[NSPasteboardTypeURL]];
	}
	return self
}

//- (void)dealloc
//{
//    [trackingArea release];
//    [markedText release];
//    [super dealloc];
//}
//
//- (BOOL)isOpaque
//{
//    return [window->ns.object isOpaque];
//}
//

func (v *contentView) CanBecomeKeyView(cmd objc.SEL) bool {
	return true
}

func (v *contentView) AcceptsFirstResponder(self objc.ID, cmd objc.SEL) bool {
	return true
}

func (v *contentView) WantsUpdateLayer(cmd objc.SEL) bool {
	return true
}

// - (void)updateLayer
//
//	{
//	   if (window->context.source == GLFW_NATIVE_CONTEXT_API)
//	       [window->context.nsgl.object update];
//
//	   _glfwInputWindowDamage(window);
//	}
//
// - (void)cursorUpdate:(NSEvent *)event
//
//	{
//	   updateCursorImage(window);
//	}
//
// - (BOOL)acceptsFirstMouse:(NSEvent *)event
//
//	{
//	   return YES;
//	}

func (v *contentView) MouseDown(_ objc.SEL, event objc.ID) {
	v.window.inputMouseClick(MouseButtonLeft, Press, translateFlags(cocoa.NSEventModifierFlags(cocoa.NSEvent{ID: event}.ModifierFlags())))
}

func (v *contentView) MouseDragged(_ objc.SEL, event objc.ID) {
	objc.ID(unsafe.Pointer(v)).Send(objc.RegisterName("mouseMoved:"), event)
}

func (v *contentView) MouseUp(_ objc.SEL, event objc.ID) {
	v.window.inputMouseClick(MouseButtonLeft, Release, translateFlags(cocoa.NSEventModifierFlags(cocoa.NSEvent{ID: event}.ModifierFlags())))
}

func (v *contentView) MouseMoved(_ objc.SEL, event objc.ID) {
	fmt.Println("FIXME: MouseMoved")
	//    if (window->cursorMode == GLFW_CURSOR_DISABLED)
	//    {
	//        const double dx = [event deltaX] - window->ns.cursorWarpDeltaX;
	//        const double dy = [event deltaY] - window->ns.cursorWarpDeltaY;
	//
	//        _glfwInputCursorPos(window,
	//                            window->virtualCursorPosX + dx,
	//                            window->virtualCursorPosY + dy);
	//    }
	//    else
	//    {
	//        const NSRect contentRect = [window->ns.view frame];
	//        // NOTE: The returned location uses base 0,1 not 0,0
	//        const NSPoint pos = [event locationInWindow];
	//
	//        _glfwInputCursorPos(window, pos.x, contentRect.size.height - pos.y);
	//    }
	//
	//    window->ns.cursorWarpDeltaX = 0;
	//    window->ns.cursorWarpDeltaY = 0;
}

//- (void)rightMouseDown:(NSEvent *)event
//{
//    _glfwInputMouseClick(window,
//                         GLFW_MOUSE_BUTTON_RIGHT,
//                         GLFW_PRESS,
//                         translateFlags([event modifierFlags]));
//}
//
//- (void)rightMouseDragged:(NSEvent *)event
//{
//    [self mouseMoved:event];
//}
//
//- (void)rightMouseUp:(NSEvent *)event
//{
//    _glfwInputMouseClick(window,
//                         GLFW_MOUSE_BUTTON_RIGHT,
//                         GLFW_RELEASE,
//                         translateFlags([event modifierFlags]));
//}
//
//- (void)otherMouseDown:(NSEvent *)event
//{
//    _glfwInputMouseClick(window,
//                         (int) [event buttonNumber],
//                         GLFW_PRESS,
//                         translateFlags([event modifierFlags]));
//}
//
//- (void)otherMouseDragged:(NSEvent *)event
//{
//    [self mouseMoved:event];
//}
//
//- (void)otherMouseUp:(NSEvent *)event
//{
//    _glfwInputMouseClick(window,
//                         (int) [event buttonNumber],
//                         GLFW_RELEASE,
//                         translateFlags([event modifierFlags]));
//}
//
//- (void)mouseExited:(NSEvent *)event
//{
//    if (window->cursorMode == GLFW_CURSOR_HIDDEN)
//        showCursor(window);
//
//    _glfwInputCursorEnter(window, GLFW_FALSE);
//}
//
//- (void)mouseEntered:(NSEvent *)event
//{
//    if (window->cursorMode == GLFW_CURSOR_HIDDEN)
//        hideCursor(window);
//
//    _glfwInputCursorEnter(window, GLFW_TRUE);
//}
//
//- (void)viewDidChangeBackingProperties
//{
//    const NSRect contentRect = [window->ns.view frame];
//    const NSRect fbRect = [window->ns.view convertRectToBacking:contentRect];
//    const float xscale = fbRect.size.width / contentRect.size.width;
//    const float yscale = fbRect.size.height / contentRect.size.height;
//
//    if (xscale != window->ns.xscale || yscale != window->ns.yscale)
//    {
//        if (window->ns.retina && window->ns.layer)
//            [window->ns.layer setContentsScale:[window->ns.object backingScaleFactor]];
//
//        window->ns.xscale = xscale;
//        window->ns.yscale = yscale;
//        _glfwInputWindowContentScale(window, xscale, yscale);
//    }
//
//    if (fbRect.size.width != window->ns.fbWidth ||
//        fbRect.size.height != window->ns.fbHeight)
//    {
//        window->ns.fbWidth  = fbRect.size.width;
//        window->ns.fbHeight = fbRect.size.height;
//        _glfwInputFramebufferSize(window, fbRect.size.width, fbRect.size.height);
//    }
//}
//
//- (void)drawRect:(NSRect)rect
//{
//    _glfwInputWindowDamage(window);
//}
//
//- (void)updateTrackingAreas
//{
//    if (trackingArea != nil)
//    {
//        [self removeTrackingArea:trackingArea];
//        [trackingArea release];
//    }
//
//    const NSTrackingAreaOptions options = NSTrackingMouseEnteredAndExited |
//                                          NSTrackingActiveInKeyWindow |
//                                          NSTrackingEnabledDuringMouseDrag |
//                                          NSTrackingCursorUpdate |
//                                          NSTrackingInVisibleRect |
//                                          NSTrackingAssumeInside;
//
//    trackingArea = [[NSTrackingArea alloc] initWithRect:[self bounds]
//                                                options:options
//                                                  owner:self
//                                               userInfo:nil];
//
//    [self addTrackingArea:trackingArea];
//    [super updateTrackingAreas];
//}

func (v *contentView) KeyDown(cmd objc.SEL, event objc.ID) {
	e := cocoa.NSEvent{event}
	key := translateKey(e.KeyCode())
	mods := translateFlags(cocoa.NSEventModifierFlags(e.ModifierFlags()))

	v.window.inputKey(key, int(e.KeyCode()), Press, mods)
	fmt.Println("FIXME: KeyDown")
	//(objc.ID)(unsafe.Pointer(v)).Send(objc.RegisterName("interpretKeyEvents:"), event)
}

// - (void)flagsChanged:(NSEvent *)event
//
//	{
//	   int action;
//	   const unsigned int modifierFlags =
//	       [event modifierFlags] & NSEventModifierFlagDeviceIndependentFlagsMask;
//	   const int key = translateKey([event keyCode]);
//	   const int mods = translateFlags(modifierFlags);
//	   const NSUInteger keyFlag = translateKeyToModifierFlag(key);
//
//	   if (keyFlag & modifierFlags)
//	   {
//	       if (window->keys[key] == GLFW_PRESS)
//	           action = GLFW_RELEASE;
//	       else
//	           action = GLFW_PRESS;
//	   }
//	   else
//	       action = GLFW_RELEASE;
//
//	   _glfwInputKey(window, key, [event keyCode], action, mods);
//	}

func (v *contentView) KeyUp(cmd objc.SEL, event objc.ID) {
	e := cocoa.NSEvent{event}
	key := translateKey(e.KeyCode())
	mods := translateFlags(cocoa.NSEventModifierFlags(e.ModifierFlags()))
	v.window.inputKey(key, int(e.KeyCode()), Release, mods)
}

//
//- (void)scrollWheel:(NSEvent *)event
//{
//    double deltaX = [event scrollingDeltaX];
//    double deltaY = [event scrollingDeltaY];
//
//    if ([event hasPreciseScrollingDeltas])
//    {
//        deltaX *= 0.1;
//        deltaY *= 0.1;
//    }
//
//    if (fabs(deltaX) > 0.0 || fabs(deltaY) > 0.0)
//        _glfwInputScroll(window, deltaX, deltaY);
//}
//
//- (NSDragOperation)draggingEntered:(id <NSDraggingInfo>)sender
//{
//    // HACK: We don't know what to say here because we don't know what the
//    //       application wants to do with the paths
//    return NSDragOperationGeneric;
//}
//
//- (BOOL)performDragOperation:(id <NSDraggingInfo>)sender
//{
//    const NSRect contentRect = [window->ns.view frame];
//    // NOTE: The returned location uses base 0,1 not 0,0
//    const NSPoint pos = [sender draggingLocation];
//    _glfwInputCursorPos(window, pos.x, contentRect.size.height - pos.y);
//
//    NSPasteboard* pasteboard = [sender draggingPasteboard];
//    NSDictionary* options = @{NSPasteboardURLReadingFileURLsOnlyKey:@YES};
//    NSArray* urls = [pasteboard readObjectsForClasses:@[[NSURL class]]
//                                              options:options];
//    const NSUInteger count = [urls count];
//    if (count)
//    {
//        char** paths = _glfw_calloc(count, sizeof(char*));
//
//        for (NSUInteger i = 0;  i < count;  i++)
//            paths[i] = _glfw_strdup([urls[i] fileSystemRepresentation]);
//
//        _glfwInputDrop(window, (int) count, (const char**) paths);
//
//        for (NSUInteger i = 0;  i < count;  i++)
//            _glfw_free(paths[i]);
//        _glfw_free(paths);
//    }
//
//    return YES;
//}
//
//- (BOOL)hasMarkedText
//{
//    return [markedText length] > 0;
//}
//
//- (NSRange)markedRange
//{
//    if ([markedText length] > 0)
//        return NSMakeRange(0, [markedText length] - 1);
//    else
//        return kEmptyRange;
//}
//
//- (NSRange)selectedRange
//{
//    return kEmptyRange;
//}
//
//- (void)setMarkedText:(id)string
//        selectedRange:(NSRange)selectedRange
//     replacementRange:(NSRange)replacementRange
//{
//    [markedText release];
//    if ([string isKindOfClass:[NSAttributedString class]])
//        markedText = [[NSMutableAttributedString alloc] initWithAttributedString:string];
//    else
//        markedText = [[NSMutableAttributedString alloc] initWithString:string];
//}
//
//- (void)unmarkText
//{
//    [[markedText mutableString] setString:@""];
//}
//

func (v *contentView) ValidAttributesForMarkedText(cmd objc.SEL) objc.ID {
	return cocoa.NSArray_array().ID
}

//- (NSAttributedString*)attributedSubstringForProposedRange:(NSRange)range
//                                               actualRange:(NSRangePointer)actualRange
//{
//    return nil;
//}
//
//- (NSUInteger)characterIndexForPoint:(NSPoint)point
//{
//    return 0;
//}
//
//- (NSRect)firstRectForCharacterRange:(NSRange)range
//                         actualRange:(NSRangePointer)actualRange
//{
//    const NSRect frame = [window->ns.view frame];
//    return NSMakeRect(frame.origin.x, frame.origin.y, 0.0, 0.0);
//}
//
//- (void)insertText:(id)string replacementRange:(NSRange)replacementRange
//{
//    NSString* characters;
//    NSEvent* event = [NSApp currentEvent];
//    const int mods = translateFlags([event modifierFlags]);
//    const int plain = !(mods & GLFW_MOD_SUPER);
//
//    if ([string isKindOfClass:[NSAttributedString class]])
//        characters = [string string];
//    else
//        characters = (NSString*) string;
//
//    NSRange range = NSMakeRange(0, [characters length]);
//    while (range.length)
//    {
//        uint32_t codepoint = 0;
//
//        if ([characters getBytes:&codepoint
//                       maxLength:sizeof(codepoint)
//                      usedLength:NULL
//                        encoding:NSUTF32StringEncoding
//                         options:0
//                           range:range
//                  remainingRange:&range])
//        {
//            if (codepoint >= 0xf700 && codepoint <= 0xf7ff)
//                continue;
//
//            _glfwInputChar(window, codepoint, mods, plain);
//        }
//    }
//}
//
//- (void)doCommandBySelector:(SEL)selector
//{
//}
//
//@end

func (v *contentView) Selector(s string) objc.SEL {
	switch s {
	case "InitWithGlfwWindow":
		return sel_initWithGlfwWindow
	case "CanBecomeKeyView":
		return objc.RegisterName("canBecomeKeyView")
	case "AcceptsFirstResponder":
		return objc.RegisterName("acceptsFirstResponder")
	case "WantsUpdateLayer":
		return objc.RegisterName("wantsUpdateLayer")
	case "ValidAttributesForMarkedText":
		return objc.RegisterName("validAttributesForMarkedText")
	case "KeyDown":
		return objc.RegisterName("keyDown:")
	case "KeyUp":
		return objc.RegisterName("keyUp:")
	case "MouseDown":
		return objc.RegisterName("mouseDown:")
	case "MouseUp":
		return objc.RegisterName("mouseUp:")
	case "MouseDragged":
		return objc.RegisterName("mouseDragged:")
	case "MouseMoved":
		return objc.RegisterName("mouseMoved:")
	default:
		return 0
	}
}

func init() {
	var err error
	class_GLFWWindowDelegate, err = objc.RegisterClass(&windowDelegate{})
	if err != nil {
		panic(err)
	}
	class_GLFWWindow, err = objc.RegisterClass(&window{})
	if err != nil {
		panic(err)
	}
	class_GLFWContentView, err = objc.RegisterClass(&contentView{})
	if err != nil {
		panic(err)
	}
}

func platformGetScancodeName(scancode int) (string, error) {
	panic("TODO: platformGetScancodeName")
	return "", nil
}

func platformGetKeyScancode(key Key) int {
	return _glfw.state.scancodes[key]
}

func (w *Window) createNativeWindow(wndconfig *wndconfig, fbconfig *fbconfig) error {
	w.state.delegate = cocoa.NSObject_alloc(class_GLFWWindowDelegate).Send(sel_initWithGlfwWindow, unsafe.Pointer(w))
	if w.state.delegate == 0 {
		return fmt.Errorf("cocoa: failed to create window delegate")
	}
	var contentRect cocoa.NSRect
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
		if wndconfig.xpos == AnyPosition || wndconfig.ypos == AnyPosition {
			contentRect = cocoa.NSRect{Origin: cocoa.CGPoint{}, Size: cocoa.NSSize{Width: cocoa.CGFloat(wndconfig.width), Height: cocoa.CGFloat(wndconfig.height)}}
		} else {
			xpos := wndconfig.xpos
			ypos := transformYCocoa(float32(wndconfig.ypos + wndconfig.height - 1))
			contentRect = cocoa.NSMakeRect(cocoa.CGFloat(xpos), cocoa.CGFloat(ypos), cocoa.CGFloat(wndconfig.width), cocoa.CGFloat(wndconfig.height))
		}
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
	w.state.object = cocoa.NSWindow{ID: cocoa.NSObject_alloc(class_GLFWWindow)}.InitWithContentRectStyleMaskBackingDefer(contentRect, styleMask, cocoa.NSBackingStoreBuffered, false).ID

	if w.state.object == 0 {
		return fmt.Errorf("cocoa: failed to create window")
	}
	nsWindow := cocoa.NSWindow{ID: w.state.object}

	if w.monitor != nil {
		//        [window->ns.object setLevel:NSMainMenuWindowLevel + 1];
		panic("TODO:")
	} else {
		fmt.Println("FIXME:")
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
	}
	//
	//    if (strlen(wndconfig->ns.frameName))
	//        [window->ns.object setFrameAutosaveName:@(wndconfig->ns.frameName)];
	//
	w.state.view = cocoa.NSObject_alloc(class_GLFWContentView).Send(sel_initWithGlfwWindow, w)
	//    window->ns.retina = wndconfig->ns.retina;

	if fbconfig.transparent {
		//        [window->ns.object setOpaque:NO];
		//        [window->ns.object setHasShadow:NO];
		//        [window->ns.object setBackgroundColor:[NSColor clearColor]];
		panic("TODO:")
	}
	nsWindow.SetContentView(w.state.view)
	nsWindow.SetDelegate(w.state.delegate)
	nsWindow.SetTitle(cocoa.NSString_alloc().InitWithUTF8String(wndconfig.title))
	//    [window->ns.object makeFirstResponder:window->ns.view];
	//    [window->ns.object setAcceptsMouseMovedEvents:YES];
	//    [window->ns.object setRestorable:NO];
	//
	//#if MAC_OS_X_VERSION_MAX_ALLOWED >= 101200
	//    if ([window->ns.object respondsToSelector:@selector(setTabbingMode:)])
	//        [window->ns.object setTabbingMode:NSWindowTabbingModeDisallowed];
	//#endif
	//
	// w.state.width, w.state.height, _ = w.platformGetWindowSize()
	// w.state.fbWidth, w.state.fbHeight, _ = w.platformGetFramebufferSize()
	return nil
}

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	if err := w.createNativeWindow(wndconfig, fbconfig); err != nil {
		return err
	}
	if ctxconfig.client != NoAPI {
		switch ctxconfig.source {
		case NativeContextAPI:
			if err := initNSGL(); err != nil {
				return err
			}
			if err := createContextNSGL(w, ctxconfig, fbconfig); err != nil {
				return err
			}
		default:
			panic("cocoa: implement context")
		}

		//        if (ctxconfig->source == GLFW_NATIVE_CONTEXT_API)
		//        {
		//            if (!_glfwInitNSGL())
		//                return GLFW_FALSE;
		//            if (!_glfwCreateContextNSGL(window, ctxconfig, fbconfig))
		//                return GLFW_FALSE;
		//        }
		//        else if (ctxconfig->source == GLFW_EGL_CONTEXT_API)
		//        {
		//            // EGL implementation on macOS use CALayer* EGLNativeWindowType so we
		//            // need to get the layer for EGL window surface creation.
		//            [window->ns.view setWantsLayer:YES];
		//            window->ns.layer = [window->ns.view layer];
		//
		//            if (!_glfwInitEGL())
		//                return GLFW_FALSE;
		//            if (!_glfwCreateContextEGL(window, ctxconfig, fbconfig))
		//                return GLFW_FALSE;
		//        }
		//        else if (ctxconfig->source == GLFW_OSMESA_CONTEXT_API)
		//        {
		//            if (!_glfwInitOSMesa())
		//                return GLFW_FALSE;
		//            if (!_glfwCreateContextOSMesa(window, ctxconfig, fbconfig))
		//                return GLFW_FALSE;
		//        }
		//
		//        if (!_glfwRefreshContextAttribs(window, ctxconfig))
		//            return GLFW_FALSE;
	}

	//    if (wndconfig->mousePassthrough)
	//        _glfwSetWindowMousePassthroughCocoa(window, GLFW_TRUE);

	if w.monitor != nil {
		w.platformShowWindow()
		_ = w.platformFocusWindow() // focus window can't return an error
		//acquireMonitor(window);
		//
		//if (wndconfig->centerCursor)
		//    _glfwCenterCursorInContentArea(window);
	} else {
		if wndconfig.visible {
			w.platformShowWindow()
			if wndconfig.focused {
				_ = w.platformFocusWindow() // focus can't return error
			}
		}
	}
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
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	event := cocoa.NSEvent_otherEventWithTypeLocationModifierFlagsTimestampWindowNumberContextSubtypeData1Data2(
		cocoa.NSEventTypeApplicationDefined,
		cocoa.NSMakePoint(0, 0),
		0, 0, 0, 0, 0, 0, 0)
	cocoa.NSApp.PostEventAtStart(event, true)
	return nil
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
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	sel_contentRectForFrameRect := objc.RegisterName("contentRectForFrameRect:")
	sig := cocoa.NSMethodSignature_signatureWithObjCTypes("{NSRect=dddd}@:{NSRect=dddd}")
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_contentRectForFrameRect)
	frame := cocoa.NSWindow{w.state.object}.Frame()
	inv.SetArgumentAtIndex(unsafe.Pointer(&frame), 2)
	inv.InvokeWithTarget(w.state.object)
	var contentRect cocoa.NSRect
	inv.GetReturnValue(unsafe.Pointer(&contentRect))
	return int(contentRect.Origin.X), int(transformYCocoa(float32(contentRect.Origin.Y + contentRect.Size.Height - 1))), nil

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
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	contentRect := cocoa.NSView{ID: w.state.view}.Frame()
	// NOTE: The returned location uses base 0,1 not 0,0
	pos := cocoa.NSWindow{ID: w.state.object}.MouseLocationOutsideOfEventStream()
	return pos.X, contentRect.Size.Height - pos.Y, nil
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
	if !_glfw.initialized {
		return 0, NotInitialized
	}
	return uintptr(w.state.object), nil
}

// Transforms a y-coordinate between the CG display and NS screen spaces
func transformYCocoa(y float32) float32 {
	return float32(C.CGDisplayBounds(C.CGMainDisplayID()).size.height - C.double(y) - 1)
}

// Translates macOS key modifiers into GLFW ones
func translateFlags(flags cocoa.NSEventModifierFlags) ModifierKey {
	var mods ModifierKey = 0

	if flags&cocoa.NSEventModifierFlagShift != 0 {
		mods |= ModShift
	}
	if flags&cocoa.NSEventModifierFlagControl != 0 {
		mods |= ModControl
	}
	if flags&cocoa.NSEventModifierFlagOption != 0 {
		mods |= ModAlt
	}
	if flags&cocoa.NSEventModifierFlagCommand != 0 {
		mods |= ModSuper
	}
	if flags&cocoa.NSEventModifierFlagCapsLock != 0 {
		mods |= ModCapsLock
	}
	return mods
}

// Translates a macOS keycode to a GLFW keycode
func translateKey(key uint16) Key {
	if key >= uint16(len(_glfw.state.keycodes)) {
		return KeyUnknown
	}
	return _glfw.state.keycodes[key]
}

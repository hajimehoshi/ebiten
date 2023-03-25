// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

import (
	"fmt"
	"log"

	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

var (
	class_GLFWHelper              objc.Class
	class_GLFWApplicationDelegate objc.Class
)
var sel_doNothing = objc.RegisterName("doNothing:")

func init() {
	class_GLFWHelper = objc.AllocateClassPair(objc.GetClass("NSObject"), "GLFWHelper", 0)
	// TODO: implement this
	//- (void)selectedKeyboardInputSourceChanged:(NSObject* )object
	//{
	//    updateUnicodeData();
	//}
	class_GLFWHelper.AddMethod(sel_doNothing, objc.NewIMP(func(self objc.ID, cmd objc.SEL, object objc.ID) {
		// does nothing :)
	}), "v@:@")
	class_GLFWHelper.Register()
	class_GLFWApplicationDelegate = objc.AllocateClassPair(objc.GetClass("NSObject"), "GLFWApplicationDelegate", 0)
	class_GLFWApplicationDelegate.AddProtocol(objc.GetProtocol("NSApplicationDelegate"))
	// TODO: implement these
	//- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender
	//{
	//    for (_GLFWwindow* window = _glfw.windowListHead;  window;  window = window->next)
	//        _glfwInputWindowCloseRequest(window);
	//
	//    return NSTerminateCancel;
	//}
	//
	//- (void)applicationDidChangeScreenParameters:(NSNotification *) notification
	//{
	//    for (_GLFWwindow* window = _glfw.windowListHead;  window;  window = window->next)
	//    {
	//        if (window->context.client != NO_API)
	//            [window->context.nsgl.object update];
	//    }
	//
	//    _glfwPollMonitorsCocoa();
	//}
	//
	//- (void)applicationWillFinishLaunching:(NSNotification *)notification
	//{
	//    if (_glfw.hints.init.state.menubar)
	//    {
	//        // Menu bar setup must go between sharedApplication and finishLaunching
	//        // in order to properly emulate the behavior of NSApplicationMain
	//
	//        if ([[NSBundle mainBundle] pathForResource:@"MainMenu" ofType:@"nib"])
	//        {
	//            [[NSBundle mainBundle] loadNibNamed:@"MainMenu"
	//                                          owner:NSApp
	//                                topLevelObjects:&_glfw.state.nibObjects];
	//        }
	//        else
	//            createMenuBar();
	//    }
	//}
	class_GLFWApplicationDelegate.AddMethod(objc.RegisterName("applicationDidFinishLaunching:"), objc.NewIMP(func(self objc.ID, cmd objc.ID, _ objc.ID) {
		// _glfwPostEmptyEventCocoa();
		_ = platformPostEmptyEvent() // cannot return error
		cocoa.NSApp.Stop(0)
	}), "v@:@")
	//- (void)applicationDidHide:(NSNotification *)notification
	//{
	//    for (int i = 0;  i < _glfw.monitorCount;  i++)
	//        _glfwRestoreVideoModeCocoa(_glfw.monitors[i]);
	//}
	class_GLFWApplicationDelegate.Register()
}

func platformInit() error {
	pool := cocoa.NSAutoreleasePool_new()
	_glfw.platformWindow.helper = cocoa.NSObject_new(class_GLFWHelper)
	cocoa.NSThread_detachNewThreadSelectorToTargetWithObject(sel_doNothing, _glfw.platformWindow.helper, 0)

	cocoa.NSApplication_sharedApplication()

	_glfw.platformWindow.delegate = cocoa.NSObject_new(class_GLFWApplicationDelegate)
	if _glfw.platformWindow.delegate == 0 {
		return fmt.Errorf("cocoa: failed to create application delegate")
	}

	cocoa.NSApp.SetDelegate(_glfw.platformWindow.delegate)
	log.Println("platformInit: todo...")

	//    NSEvent* (^block)(NSEvent*) = ^ NSEvent* (NSEvent* event)
	//    {
	//        if ([event modifierFlags] & NSEventModifierFlagCommand)
	//            [[NSApp keyWindow] sendEvent:event];
	//
	//        return event;
	//    };
	//
	//    _glfw.state.keyUpMonitor =
	//        [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyUp
	//                                              handler:block];
	//
	//    if (_glfw.hints.init.state.chdir)
	//        changeToResourcesDirectory();
	//
	//    // Press and Hold prevents some keys from emitting repeated characters
	//    NSDictionary* defaults = @{@"ApplePressAndHoldEnabled":@NO};
	//    [[NSUserDefaults standardUserDefaults] registerDefaults:defaults];
	//
	//    [[NSNotificationCenter defaultCenter]
	//        addObserver:_glfw.state.helper
	//           selector:@selector(selectedKeyboardInputSourceChanged:)
	//               name:NSTextInputContextKeyboardSelectionDidChangeNotification
	//             object:nil];

	createKeyTables()

	//    _glfw.state.eventSource = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	//    if (!_glfw.state.eventSource)
	//        return FALSE;
	//
	//    CGEventSourceSetLocalEventsSuppressionInterval(_glfw.state.eventSource, 0.0);
	//
	//    if (!initializeTIS())
	//        return FALSE;
	//
	err := platformPollMonitors()
	if err != nil {
		return err
	}
	if !cocoa.NSRunningApplication_currentApplication().IsFinishedLaunching() {
		cocoa.NSApp.Run()
	}

	// In case we are unbundled, make us a proper UI application
	if !_glfw.hints.init.ns.menubar {
		cocoa.NSApp.SetActivationPolicy(cocoa.NSApplicationActivationPolicyRegular)
	}
	pool.Release()
	return nil
}

func platformTerminate() error {
	log.Println("glfw: platformTerminate: NOT IMPLEMENTED")
	return nil
	//@autoreleasepool {
	//
	//    if (_glfw.state.inputSource)
	//    {
	//        CFRelease(_glfw.state.inputSource);
	//        _glfw.state.inputSource = NULL;
	//        _glfw.state.unicodeData = nil;
	//    }
	//
	//    if (_glfw.state.eventSource)
	//    {
	//        CFRelease(_glfw.state.eventSource);
	//        _glfw.state.eventSource = NULL;
	//    }
	//
	//    if (_glfw.state.delegate)
	//    {
	//        [NSApp setDelegate:nil];
	//        [_glfw.state.delegate release];
	//        _glfw.state.delegate = nil;
	//    }
	//
	//    if (_glfw.state.helper)
	//    {
	//        [[NSNotificationCenter defaultCenter]
	//            removeObserver:_glfw.state.helper
	//                      name:NSTextInputContextKeyboardSelectionDidChangeNotification
	//                    object:nil];
	//        [[NSNotificationCenter defaultCenter]
	//            removeObserver:_glfw.state.helper];
	//        [_glfw.state.helper release];
	//        _glfw.state.helper = nil;
	//    }
	//
	//    if (_glfw.state.keyUpMonitor)
	//        [NSEvent removeMonitor:_glfw.state.keyUpMonitor];
	//
	//    _glfw_free(_glfw.state.clipboardString);
	//
	//    _glfwTerminateNSGL();
	//    _glfwTerminateEGL();
	//    _glfwTerminateOSMesa();
	//
	//    } // autoreleasepool
}

// Create key code translation tables
func createKeyTables() {
	for i := range _glfw.platformWindow.keycodes {
		_glfw.platformWindow.keycodes[i] = -1
	}
	for i := range _glfw.platformWindow.scancodes {
		_glfw.platformWindow.scancodes[i] = -1
	}

	_glfw.platformWindow.keycodes[0x1D] = Key0
	_glfw.platformWindow.keycodes[0x12] = Key1
	_glfw.platformWindow.keycodes[0x13] = Key2
	_glfw.platformWindow.keycodes[0x14] = Key3
	_glfw.platformWindow.keycodes[0x15] = Key4
	_glfw.platformWindow.keycodes[0x17] = Key5
	_glfw.platformWindow.keycodes[0x16] = Key6
	_glfw.platformWindow.keycodes[0x1A] = Key7
	_glfw.platformWindow.keycodes[0x1C] = Key8
	_glfw.platformWindow.keycodes[0x19] = Key9
	_glfw.platformWindow.keycodes[0x00] = KeyA
	_glfw.platformWindow.keycodes[0x0B] = KeyB
	_glfw.platformWindow.keycodes[0x08] = KeyC
	_glfw.platformWindow.keycodes[0x02] = KeyD
	_glfw.platformWindow.keycodes[0x0E] = KeyE
	_glfw.platformWindow.keycodes[0x03] = KeyF
	_glfw.platformWindow.keycodes[0x05] = KeyG
	_glfw.platformWindow.keycodes[0x04] = KeyH
	_glfw.platformWindow.keycodes[0x22] = KeyI
	_glfw.platformWindow.keycodes[0x26] = KeyJ
	_glfw.platformWindow.keycodes[0x28] = KeyK
	_glfw.platformWindow.keycodes[0x25] = KeyL
	_glfw.platformWindow.keycodes[0x2E] = KeyM
	_glfw.platformWindow.keycodes[0x2D] = KeyN
	_glfw.platformWindow.keycodes[0x1F] = KeyO
	_glfw.platformWindow.keycodes[0x23] = KeyP
	_glfw.platformWindow.keycodes[0x0C] = KeyQ
	_glfw.platformWindow.keycodes[0x0F] = KeyR
	_glfw.platformWindow.keycodes[0x01] = KeyS
	_glfw.platformWindow.keycodes[0x11] = KeyT
	_glfw.platformWindow.keycodes[0x20] = KeyU
	_glfw.platformWindow.keycodes[0x09] = KeyV
	_glfw.platformWindow.keycodes[0x0D] = KeyW
	_glfw.platformWindow.keycodes[0x07] = KeyX
	_glfw.platformWindow.keycodes[0x10] = KeyY
	_glfw.platformWindow.keycodes[0x06] = KeyZ

	_glfw.platformWindow.keycodes[0x27] = KeyApostrophe
	_glfw.platformWindow.keycodes[0x2A] = KeyBackslash
	_glfw.platformWindow.keycodes[0x2B] = KeyComma
	_glfw.platformWindow.keycodes[0x18] = KeyEqual
	_glfw.platformWindow.keycodes[0x32] = KeyGraveAccent
	_glfw.platformWindow.keycodes[0x21] = KeyLeftBracket
	_glfw.platformWindow.keycodes[0x1B] = KeyMinus
	_glfw.platformWindow.keycodes[0x2F] = KeyPeriod
	_glfw.platformWindow.keycodes[0x1E] = KeyRightBracket
	_glfw.platformWindow.keycodes[0x29] = KeySemicolon
	_glfw.platformWindow.keycodes[0x2C] = KeySlash
	_glfw.platformWindow.keycodes[0x0A] = KeyWorld1

	_glfw.platformWindow.keycodes[0x33] = KeyBackspace
	_glfw.platformWindow.keycodes[0x39] = KeyCapsLock
	_glfw.platformWindow.keycodes[0x75] = KeyDelete
	_glfw.platformWindow.keycodes[0x7D] = KeyDown
	_glfw.platformWindow.keycodes[0x77] = KeyEnd
	_glfw.platformWindow.keycodes[0x24] = KeyEnter
	_glfw.platformWindow.keycodes[0x35] = KeyEscape
	_glfw.platformWindow.keycodes[0x7A] = KeyF1
	_glfw.platformWindow.keycodes[0x78] = KeyF2
	_glfw.platformWindow.keycodes[0x63] = KeyF3
	_glfw.platformWindow.keycodes[0x76] = KeyF4
	_glfw.platformWindow.keycodes[0x60] = KeyF5
	_glfw.platformWindow.keycodes[0x61] = KeyF6
	_glfw.platformWindow.keycodes[0x62] = KeyF7
	_glfw.platformWindow.keycodes[0x64] = KeyF8
	_glfw.platformWindow.keycodes[0x65] = KeyF9
	_glfw.platformWindow.keycodes[0x6D] = KeyF10
	_glfw.platformWindow.keycodes[0x67] = KeyF11
	_glfw.platformWindow.keycodes[0x6F] = KeyF12
	_glfw.platformWindow.keycodes[0x69] = KeyPrintScreen
	_glfw.platformWindow.keycodes[0x6B] = KeyF14
	_glfw.platformWindow.keycodes[0x71] = KeyF15
	_glfw.platformWindow.keycodes[0x6A] = KeyF16
	_glfw.platformWindow.keycodes[0x40] = KeyF17
	_glfw.platformWindow.keycodes[0x4F] = KeyF18
	_glfw.platformWindow.keycodes[0x50] = KeyF19
	_glfw.platformWindow.keycodes[0x5A] = KeyF20
	_glfw.platformWindow.keycodes[0x73] = KeyHome
	_glfw.platformWindow.keycodes[0x72] = KeyInsert
	_glfw.platformWindow.keycodes[0x7B] = KeyLeft
	_glfw.platformWindow.keycodes[0x3A] = KeyLeftAlt
	_glfw.platformWindow.keycodes[0x3B] = KeyLeftControl
	_glfw.platformWindow.keycodes[0x38] = KeyLeftShift
	_glfw.platformWindow.keycodes[0x37] = KeyLeftSuper
	_glfw.platformWindow.keycodes[0x6E] = KeyMenu
	_glfw.platformWindow.keycodes[0x47] = KeyNumLock
	_glfw.platformWindow.keycodes[0x79] = KeyPageDown
	_glfw.platformWindow.keycodes[0x74] = KeyPageUp
	_glfw.platformWindow.keycodes[0x7C] = KeyRight
	_glfw.platformWindow.keycodes[0x3D] = KeyRightAlt
	_glfw.platformWindow.keycodes[0x3E] = KeyRightControl
	_glfw.platformWindow.keycodes[0x3C] = KeyRightShift
	_glfw.platformWindow.keycodes[0x36] = KeyRightSuper
	_glfw.platformWindow.keycodes[0x31] = KeySpace
	_glfw.platformWindow.keycodes[0x30] = KeyTab
	_glfw.platformWindow.keycodes[0x7E] = KeyUp

	_glfw.platformWindow.keycodes[0x52] = KeyKP0
	_glfw.platformWindow.keycodes[0x53] = KeyKP1
	_glfw.platformWindow.keycodes[0x54] = KeyKP2
	_glfw.platformWindow.keycodes[0x55] = KeyKP3
	_glfw.platformWindow.keycodes[0x56] = KeyKP4
	_glfw.platformWindow.keycodes[0x57] = KeyKP5
	_glfw.platformWindow.keycodes[0x58] = KeyKP6
	_glfw.platformWindow.keycodes[0x59] = KeyKP7
	_glfw.platformWindow.keycodes[0x5B] = KeyKP8
	_glfw.platformWindow.keycodes[0x5C] = KeyKP9
	_glfw.platformWindow.keycodes[0x45] = KeyKPAdd
	_glfw.platformWindow.keycodes[0x41] = KeyKPDecimal
	_glfw.platformWindow.keycodes[0x4B] = KeyKPDivide
	_glfw.platformWindow.keycodes[0x4C] = KeyKPEnter
	_glfw.platformWindow.keycodes[0x51] = KeyKPEqual
	_glfw.platformWindow.keycodes[0x43] = KeyKPMultiply
	_glfw.platformWindow.keycodes[0x4E] = KeyKPSubtract

	for scancode := 0; scancode < 256; scancode++ {
		// Store the reverse translation for faster key name lookup
		if _glfw.platformWindow.keycodes[scancode] >= 0 {
			_glfw.platformWindow.scancodes[_glfw.platformWindow.keycodes[scancode]] = scancode
		}
	}
}

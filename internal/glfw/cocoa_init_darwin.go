// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// cfString creates a CFStringRef from a Go string. The caller is responsible for releasing it.
func cfString(s string) uintptr {
	return cfStringCreateWithCString(0, s, kCFStringEncodingUTF8)
}

// createKeyTables builds the macOS virtual key code to GLFW key mapping tables.
func createKeyTables() {
	for i := range _glfw.platformWindow.keycodes {
		_glfw.platformWindow.keycodes[i] = -1
	}
	for i := range _glfw.platformWindow.scancodes {
		_glfw.platformWindow.scancodes[i] = -1
	}

	_glfw.platformWindow.keycodes[0x00] = KeyA
	_glfw.platformWindow.keycodes[0x01] = KeyS
	_glfw.platformWindow.keycodes[0x02] = KeyD
	_glfw.platformWindow.keycodes[0x03] = KeyF
	_glfw.platformWindow.keycodes[0x04] = KeyH
	_glfw.platformWindow.keycodes[0x05] = KeyG
	_glfw.platformWindow.keycodes[0x06] = KeyZ
	_glfw.platformWindow.keycodes[0x07] = KeyX
	_glfw.platformWindow.keycodes[0x08] = KeyC
	_glfw.platformWindow.keycodes[0x09] = KeyV
	_glfw.platformWindow.keycodes[0x0B] = KeyB
	_glfw.platformWindow.keycodes[0x0C] = KeyQ
	_glfw.platformWindow.keycodes[0x0D] = KeyW
	_glfw.platformWindow.keycodes[0x0E] = KeyE
	_glfw.platformWindow.keycodes[0x0F] = KeyR
	_glfw.platformWindow.keycodes[0x10] = KeyY
	_glfw.platformWindow.keycodes[0x11] = KeyT
	_glfw.platformWindow.keycodes[0x12] = Key1
	_glfw.platformWindow.keycodes[0x13] = Key2
	_glfw.platformWindow.keycodes[0x14] = Key3
	_glfw.platformWindow.keycodes[0x15] = Key4
	_glfw.platformWindow.keycodes[0x16] = Key6
	_glfw.platformWindow.keycodes[0x17] = Key5
	_glfw.platformWindow.keycodes[0x18] = KeyEqual
	_glfw.platformWindow.keycodes[0x19] = Key9
	_glfw.platformWindow.keycodes[0x1A] = Key7
	_glfw.platformWindow.keycodes[0x1B] = KeyMinus
	_glfw.platformWindow.keycodes[0x1C] = Key8
	_glfw.platformWindow.keycodes[0x1D] = Key0
	_glfw.platformWindow.keycodes[0x1E] = KeyRightBracket
	_glfw.platformWindow.keycodes[0x1F] = KeyO
	_glfw.platformWindow.keycodes[0x20] = KeyU
	_glfw.platformWindow.keycodes[0x21] = KeyLeftBracket
	_glfw.platformWindow.keycodes[0x22] = KeyI
	_glfw.platformWindow.keycodes[0x23] = KeyP
	_glfw.platformWindow.keycodes[0x24] = KeyEnter
	_glfw.platformWindow.keycodes[0x25] = KeyL
	_glfw.platformWindow.keycodes[0x26] = KeyJ
	_glfw.platformWindow.keycodes[0x27] = KeyApostrophe
	_glfw.platformWindow.keycodes[0x28] = KeyK
	_glfw.platformWindow.keycodes[0x29] = KeySemicolon
	_glfw.platformWindow.keycodes[0x2A] = KeyBackslash
	_glfw.platformWindow.keycodes[0x2B] = KeyComma
	_glfw.platformWindow.keycodes[0x2C] = KeySlash
	_glfw.platformWindow.keycodes[0x2D] = KeyN
	_glfw.platformWindow.keycodes[0x2E] = KeyM
	_glfw.platformWindow.keycodes[0x2F] = KeyPeriod
	_glfw.platformWindow.keycodes[0x30] = KeyTab
	_glfw.platformWindow.keycodes[0x31] = KeySpace
	_glfw.platformWindow.keycodes[0x32] = KeyGraveAccent
	_glfw.platformWindow.keycodes[0x33] = KeyBackspace
	_glfw.platformWindow.keycodes[0x35] = KeyEscape
	_glfw.platformWindow.keycodes[0x37] = KeyLeftSuper
	_glfw.platformWindow.keycodes[0x38] = KeyLeftShift
	_glfw.platformWindow.keycodes[0x39] = KeyCapsLock
	_glfw.platformWindow.keycodes[0x3A] = KeyLeftAlt
	_glfw.platformWindow.keycodes[0x3B] = KeyLeftControl
	_glfw.platformWindow.keycodes[0x3C] = KeyRightShift
	_glfw.platformWindow.keycodes[0x3D] = KeyRightAlt
	_glfw.platformWindow.keycodes[0x3E] = KeyRightControl
	_glfw.platformWindow.keycodes[0x3F] = KeyRightSuper
	_glfw.platformWindow.keycodes[0x40] = KeyF17
	_glfw.platformWindow.keycodes[0x43] = KeyKPDecimal
	_glfw.platformWindow.keycodes[0x45] = KeyKPMultiply
	_glfw.platformWindow.keycodes[0x47] = KeyNumLock
	_glfw.platformWindow.keycodes[0x48] = KeyKPAdd
	_glfw.platformWindow.keycodes[0x4B] = KeyKPDivide
	_glfw.platformWindow.keycodes[0x4C] = KeyKPEnter
	_glfw.platformWindow.keycodes[0x4E] = KeyKPSubtract
	_glfw.platformWindow.keycodes[0x4F] = KeyF18
	_glfw.platformWindow.keycodes[0x50] = KeyF19
	_glfw.platformWindow.keycodes[0x51] = KeyKPEqual
	_glfw.platformWindow.keycodes[0x52] = KeyKP0
	_glfw.platformWindow.keycodes[0x53] = KeyKP1
	_glfw.platformWindow.keycodes[0x54] = KeyKP2
	_glfw.platformWindow.keycodes[0x55] = KeyKP3
	_glfw.platformWindow.keycodes[0x56] = KeyKP4
	_glfw.platformWindow.keycodes[0x57] = KeyKP5
	_glfw.platformWindow.keycodes[0x58] = KeyKP6
	_glfw.platformWindow.keycodes[0x59] = KeyKP7
	_glfw.platformWindow.keycodes[0x5A] = KeyF20
	_glfw.platformWindow.keycodes[0x5B] = KeyKP8
	_glfw.platformWindow.keycodes[0x5C] = KeyKP9
	_glfw.platformWindow.keycodes[0x60] = KeyF5
	_glfw.platformWindow.keycodes[0x61] = KeyF6
	_glfw.platformWindow.keycodes[0x62] = KeyF7
	_glfw.platformWindow.keycodes[0x63] = KeyF3
	_glfw.platformWindow.keycodes[0x64] = KeyF8
	_glfw.platformWindow.keycodes[0x65] = KeyF9
	_glfw.platformWindow.keycodes[0x67] = KeyF11
	_glfw.platformWindow.keycodes[0x69] = KeyF13
	_glfw.platformWindow.keycodes[0x6A] = KeyF16
	_glfw.platformWindow.keycodes[0x6B] = KeyF14
	_glfw.platformWindow.keycodes[0x6D] = KeyF10
	_glfw.platformWindow.keycodes[0x6E] = KeyMenu
	_glfw.platformWindow.keycodes[0x6F] = KeyF12
	_glfw.platformWindow.keycodes[0x71] = KeyF15
	_glfw.platformWindow.keycodes[0x72] = KeyInsert
	_glfw.platformWindow.keycodes[0x73] = KeyHome
	_glfw.platformWindow.keycodes[0x74] = KeyPageUp
	_glfw.platformWindow.keycodes[0x75] = KeyDelete
	_glfw.platformWindow.keycodes[0x76] = KeyF4
	_glfw.platformWindow.keycodes[0x77] = KeyEnd
	_glfw.platformWindow.keycodes[0x78] = KeyF2
	_glfw.platformWindow.keycodes[0x79] = KeyPageDown
	_glfw.platformWindow.keycodes[0x7A] = KeyF1
	_glfw.platformWindow.keycodes[0x7B] = KeyLeft
	_glfw.platformWindow.keycodes[0x7C] = KeyRight
	_glfw.platformWindow.keycodes[0x7D] = KeyDown
	_glfw.platformWindow.keycodes[0x7E] = KeyUp

	for scancode := range 256 {
		if _glfw.platformWindow.keycodes[scancode] > 0 {
			_glfw.platformWindow.scancodes[_glfw.platformWindow.keycodes[scancode]] = scancode
		}
	}
}

// getAppName returns the application name from NSProcessInfo or the bundle.
func getAppName() string {
	// Try to get the name from the bundle's Info.plist first.
	bundle := objc.ID(classNSBundle).Send(selMainBundle)
	if bundle != 0 {
		info := bundle.Send(selInfoDictionary)
		if info != 0 {
			name := info.Send(selObjectForKey, cocoa.NSString_alloc().InitWithUTF8String("CFBundleName").ID)
			if name != 0 {
				s := cocoa.NSString{ID: name}.String()
				if len(s) > 0 {
					return s
				}
			}
		}
	}

	// Fall back to process name.
	pi := objc.ID(classNSProcessInfo).Send(selProcessInfo)
	name := cocoa.NSString{ID: pi.Send(selProcessName)}
	return name.String()
}

// createMenuBar creates the standard macOS menu bar with app menu and window menu.
func createMenuBar() {
	appName := getAppName()

	menubar := objc.ID(classNSMenu).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))

	// Create the application menu.
	appMenuItem := objc.ID(classNSMenuItem).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	menubar.Send(selAddItem, appMenuItem)

	appMenu := objc.ID(classNSMenu).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))

	// About <AppName>
	appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("About "+appName).ID,
		selOrderFrontStandardAboutPanel,
		cocoa.NSString_alloc().InitWithUTF8String("").ID)

	appMenu.Send(selAddItem, objc.ID(classNSMenuItem).Send(selSeparatorItem))

	// Services submenu
	servicesMenuItem := appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Services").ID,
		objc.SEL(0),
		cocoa.NSString_alloc().InitWithUTF8String("").ID)
	servicesMenu := objc.ID(classNSMenu).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	servicesMenuItem.Send(selSetSubmenu, servicesMenu)
	nsApp := objc.ID(classNSApplication).Send(selNSApp)
	nsApp.Send(selSetServicesMenu, servicesMenu)

	appMenu.Send(selAddItem, objc.ID(classNSMenuItem).Send(selSeparatorItem))

	// Hide <AppName>
	appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Hide "+appName).ID,
		selHide,
		cocoa.NSString_alloc().InitWithUTF8String("h").ID)

	// Hide Others
	hideOthersItem := appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Hide Others").ID,
		selHideOtherApplications,
		cocoa.NSString_alloc().InitWithUTF8String("h").ID)
	// NSEventModifierFlagOption | NSEventModifierFlagCommand
	hideOthersItem.Send(selSetKeyEquivalentModifierMask, uintptr(1<<19|1<<20))

	// Show All
	appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Show All").ID,
		selUnhideAllApplications,
		cocoa.NSString_alloc().InitWithUTF8String("").ID)

	appMenu.Send(selAddItem, objc.ID(classNSMenuItem).Send(selSeparatorItem))

	// Quit <AppName>
	appMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Quit "+appName).ID,
		selTerminate,
		cocoa.NSString_alloc().InitWithUTF8String("q").ID)

	appMenuItem.Send(selSetSubmenu, appMenu)

	// Create the Window menu.
	windowMenuItem := objc.ID(classNSMenuItem).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	menubar.Send(selAddItem, windowMenuItem)

	windowMenu := objc.ID(classNSMenu).Send(objc.RegisterName("alloc")).Send(
		selInitWithTitle, cocoa.NSString_alloc().InitWithUTF8String("Window").ID)

	// Minimize
	windowMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Minimize").ID,
		selMiniaturize,
		cocoa.NSString_alloc().InitWithUTF8String("m").ID)

	// Zoom
	windowMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Zoom").ID,
		selZoom,
		cocoa.NSString_alloc().InitWithUTF8String("").ID)

	windowMenu.Send(selAddItem, objc.ID(classNSMenuItem).Send(selSeparatorItem))

	// Bring All to Front
	windowMenu.Send(selAddItemWithTitle,
		cocoa.NSString_alloc().InitWithUTF8String("Bring All to Front").ID,
		selArrangeInFront,
		cocoa.NSString_alloc().InitWithUTF8String("").ID)

	windowMenuItem.Send(selSetSubmenu, windowMenu)

	nsApp.Send(selSetMainMenu, menubar)
	nsApp.Send(selSetWindowsMenu, windowMenu)
}

// updateUnicodeDataNS updates the cached keyboard layout unicode data.
func updateUnicodeDataNS() {
	inputSource := _glfw.platformWindow.tis.CopyCurrentKeyboardLayoutInputSource()
	_glfw.platformWindow.unicodeData = _glfw.platformWindow.tis.GetInputSourceProperty(
		inputSource, _glfw.platformWindow.tis.kPropertyUnicodeKeyLayoutData)
	cfRelease(inputSource)
}

// initializeTIS loads TIS (Text Input Source) symbols from the HIToolbox framework.
func initializeTIS() {
	// When using Cgo, HIToolbox is loaded implicitly by linking against Cocoa.
	// With purego, we must load it explicitly so CFBundleGetBundleWithIdentifier can find it.
	_, err := purego.Dlopen("/System/Library/Frameworks/Carbon.framework/Versions/A/Frameworks/HIToolbox.framework/HIToolbox", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Sprintf("glfw: failed to dlopen HIToolbox: %v", err))
	}

	bundleID := cfString("com.apple.HIToolbox")
	defer cfRelease(bundleID)

	bundle := cfBundleGetBundleWithIdentifier(bundleID)
	if bundle == 0 {
		panic("glfw: failed to get HIToolbox bundle")
	}

	// Load TISCopyCurrentKeyboardLayoutInputSource
	fnName1 := cfString("TISCopyCurrentKeyboardLayoutInputSource")
	defer cfRelease(fnName1)
	ptr1 := cfBundleGetFunctionPointerForName(bundle, fnName1)
	if ptr1 == 0 {
		panic("glfw: failed to load TISCopyCurrentKeyboardLayoutInputSource")
	}
	_glfw.platformWindow.tis.CopyCurrentKeyboardLayoutInputSource = func() uintptr {
		r, _, _ := purego.SyscallN(ptr1)
		return r
	}

	// Load TISGetInputSourceProperty
	fnName2 := cfString("TISGetInputSourceProperty")
	defer cfRelease(fnName2)
	ptr2 := cfBundleGetFunctionPointerForName(bundle, fnName2)
	if ptr2 == 0 {
		panic("glfw: failed to load TISGetInputSourceProperty")
	}
	_glfw.platformWindow.tis.GetInputSourceProperty = func(inputSource uintptr, propertyKey uintptr) uintptr {
		r, _, _ := purego.SyscallN(ptr2, inputSource, propertyKey)
		return r
	}

	// Load LMGetKbdType
	fnName3 := cfString("LMGetKbdType")
	defer cfRelease(fnName3)
	ptr3 := cfBundleGetFunctionPointerForName(bundle, fnName3)
	if ptr3 == 0 {
		panic("glfw: failed to load LMGetKbdType")
	}
	_glfw.platformWindow.tis.GetKbdType = func() uint8 {
		r, _, _ := purego.SyscallN(ptr3)
		return uint8(r)
	}

	// Load kTISPropertyUnicodeKeyLayoutData string constant
	dataName := cfString("kTISPropertyUnicodeKeyLayoutData")
	defer cfRelease(dataName)
	dataPtr := cfBundleGetDataPointerForName(bundle, dataName)
	if dataPtr == 0 {
		panic("glfw: failed to load kTISPropertyUnicodeKeyLayoutData")
	}
	_glfw.platformWindow.tis.kPropertyUnicodeKeyLayoutData = *(*uintptr)(unsafe.Pointer(dataPtr))
}

// GLFWHelper and GLFWApplicationDelegate class references.
var (
	classGLFWHelper              objc.Class
	classGLFWApplicationDelegate objc.Class
)

// platformInit performs the full macOS platform initialization.
func platformInit() error {
	// Register GLFWHelper class — an NSObject subclass with a method to handle
	// keyboard input source change notifications.
	helper, err := objc.RegisterClass(
		"GLFWHelper",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{
			{
				Cmd: selSelectedKeyboardInputSourceChanged,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
					updateUnicodeDataNS()
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("glfw: failed to register GLFWHelper class: %w", err)
	}
	classGLFWHelper = helper

	// Register GLFWWindow, GLFWWindowDelegate, and GLFWContentView classes.
	if err := registerGLFWClasses(); err != nil {
		return err
	}

	// Register GLFWApplicationDelegate class — implements NSApplicationDelegate.
	delegate, err := objc.RegisterClass(
		"GLFWApplicationDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("NSApplicationDelegate")},
		nil,
		[]objc.MethodDef{
			{
				Cmd: selApplicationShouldTerminate,
				Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) uintptr {
					// Post close events to all windows.
					for _, window := range _glfw.windows {
						if window.callbacks.close != nil {
							window.callbacks.close(window)
						}
					}
					return _NSApplicationTerminateCancel
				},
			},
			{
				Cmd: selApplicationDidChangeScreenParameters,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
					_ = pollMonitorsNS()
				},
			},
			{
				Cmd: selApplicationWillFinishLaunching,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
					nsApp := objc.ID(classNSApplication).Send(selNSApp)
					if nsApp.Send(selMainMenu) == 0 {
						createMenuBar()
					}
				},
			},
			{
				Cmd: selApplicationDidFinishLaunching,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
					nsApp := objc.ID(classNSApplication).Send(selNSApp)
					nsApp.Send(selStop, 0)
					// Post an empty event to ensure the run loop processes the stop.
					postEmptyEvent()
				},
			},
			{
				Cmd: selApplicationDidHide,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
					for _, window := range _glfw.windows {
						window.platform.occluded = false
					}
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("glfw: failed to register GLFWApplicationDelegate class: %w", err)
	}
	classGLFWApplicationDelegate = delegate

	// Create the shared NSApplication instance.
	nsApp := objc.ID(classNSApplication).Send(selNSApp)

	// Create and set the application delegate.
	_glfw.platformWindow.delegate = objc.ID(classGLFWApplicationDelegate).Send(
		objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	nsApp.Send(objc.RegisterName("setDelegate:"), _glfw.platformWindow.delegate)

	// Create GLFWHelper instance and register for keyboard input source change notifications.
	_glfw.platformWindow.helper = objc.ID(classGLFWHelper).Send(
		objc.RegisterName("alloc")).Send(objc.RegisterName("init"))

	notificationCenter := objc.ID(classNSNotificationCenter).Send(selDefaultCenter)
	nsTextInputContextKeyboardSelectionDidChangeNotification := cocoa.NSString_alloc().InitWithUTF8String(
		"NSTextInputContextKeyboardSelectionDidChangeNotification")
	notificationCenter.Send(selAddObserverSelectorNameObject,
		_glfw.platformWindow.helper,
		selSelectedKeyboardInputSourceChanged,
		nsTextInputContextKeyboardSelectionDidChangeNotification.ID,
		0)

	// Add a global monitor for keyUp events to work around Cocoa swallowing
	// key-up events when the menu bar is active.
	keyUpBlock := objc.NewBlock(func(_ objc.Block, event objc.ID) {
		// Re-dispatch keyUp events to the application.
		app := objc.ID(classNSApplication).Send(selNSApp)
		app.Send(selSendEvent, event)
	})
	_glfw.platformWindow.keyUpMonitor = objc.ID(classNSEvent).Send(
		selAddGlobalMonitorForEventsMatchingMask,
		_NSEventMaskKeyUp,
		keyUpBlock)

	// Create a CGEventSource for synthesized events.
	_glfw.platformWindow.eventSource = cgEventSourceCreate(_kCGEventSourceStateCombinedSessionState)

	// Initialize TIS (Text Input Source) framework bindings.
	initializeTIS()

	// Build key code translation tables.
	createKeyTables()

	// Cache the current keyboard layout unicode data.
	updateUnicodeDataNS()

	// Initialize the high-resolution timer.
	initTimerNS()

	// Detect and register connected monitors.
	if err := pollMonitorsNS(); err != nil {
		return err
	}

	// If not running from a bundle (e.g., launched from terminal),
	// set the activation policy so we get a dock icon and menu bar.
	bundle := objc.ID(classNSBundle).Send(selMainBundle)
	if bundle == 0 || bundle.Send(selBundleIdentifier) == 0 {
		nsApp.Send(selSetActivationPolicy, _NSApplicationActivationPolicyRegular)
	}

	// Run the application to process initial events. The delegate's
	// applicationDidFinishLaunching: calls stop: and posts an empty event,
	// so this returns quickly.
	nsApp.Send(selRun)

	// Initialize NSGL (OpenGL context support).
	if err := initNSGL(); err != nil {
		return err
	}

	return nil
}

// postEmptyEvent posts a no-op application-defined event to wake the run loop.
func postEmptyEvent() {
	nsApp := objc.ID(classNSApplication).Send(selNSApp)
	// NSApplicationDefined = 15
	event := objc.Send[objc.ID](objc.ID(classNSEvent), selOtherEventWithType,
		uintptr(15),         // NSApplicationDefined
		cocoa.CGPoint{0, 0}, // location (NSPoint)
		uintptr(0),          // modifierFlags
		float64(0),          // timestamp
		uintptr(0),          // windowNumber
		uintptr(0),          // context (nil)
		uintptr(0),          // subtype
		uintptr(0),          // data1
		uintptr(0),          // data2
	)
	nsApp.Send(selPostEventAtStart, event, true)
}

// platformTerminate cleans up macOS platform resources.
func platformTerminate() error {
	// Release TIS unicode data reference (if held).
	// The inputSource obtained from CopyCurrentKeyboardLayoutInputSource
	// is released in updateUnicodeDataNS, so nothing extra is needed for tis.

	// Release the CGEventSource.
	if _glfw.platformWindow.eventSource != 0 {
		cfRelease(_glfw.platformWindow.eventSource)
		_glfw.platformWindow.eventSource = 0
	}

	// Release the application delegate.
	if _glfw.platformWindow.delegate != 0 {
		nsApp := objc.ID(classNSApplication).Send(selNSApp)
		nsApp.Send(objc.RegisterName("setDelegate:"), 0)
		_glfw.platformWindow.delegate.Send(objc.RegisterName("release"))
		_glfw.platformWindow.delegate = 0
	}

	// Release the helper.
	if _glfw.platformWindow.helper != 0 {
		_glfw.platformWindow.helper.Send(objc.RegisterName("release"))
		_glfw.platformWindow.helper = 0
	}

	// Remove the global keyUp monitor.
	if _glfw.platformWindow.keyUpMonitor != 0 {
		objc.ID(classNSEvent).Send(objc.RegisterName("removeMonitor:"), _glfw.platformWindow.keyUpMonitor)
		_glfw.platformWindow.keyUpMonitor = 0
	}

	// Terminate NSGL (OpenGL context support).
	terminateNSGL()

	return nil
}

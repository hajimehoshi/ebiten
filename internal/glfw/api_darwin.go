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

package glfw

import (
	"fmt"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// CoreFoundation constants.
const (
	kCFStringEncodingUTF8 uint32 = 0x08000100
)

// NSApplication constants.
const (
	_NSApplicationActivationPolicyRegular = 0
	_NSApplicationTerminateCancel         = 0
)

// NSEvent type masks.
const (
	_NSEventMaskKeyUp = 1 << 11
)

// CGEventSource state IDs.
const (
	_kCGEventSourceStateCombinedSessionState int32 = -1
)

// NSEvent modifier flags.
const (
	NSEventModifierFlagCapsLock                   = 1 << 16
	NSEventModifierFlagShift                      = 1 << 17
	NSEventModifierFlagControl                    = 1 << 18
	NSEventModifierFlagOption                     = 1 << 19
	NSEventModifierFlagCommand                    = 1 << 20
	NSEventModifierFlagNumericPad                 = 1 << 21
	NSEventModifierFlagDeviceIndependentFlagsMask = 0xffff0000
)

// NSWindow style masks.
const (
	NSWindowStyleMaskBorderless     = 0
	NSWindowStyleMaskTitled         = 1 << 0
	NSWindowStyleMaskClosable       = 1 << 1
	NSWindowStyleMaskMiniaturizable = 1 << 2
	NSWindowStyleMaskResizable      = 1 << 3
)

// Window levels.
const (
	NSNormalWindowLevel   = 0
	NSFloatingWindowLevel = 3
	NSMainMenuWindowLevel = 24
)

// Backing store type.
const (
	NSBackingStoreBuffered = 2
)

// NSEvent types.
const (
	NSEventTypeKeyDown            = 10
	NSEventTypeKeyUp              = 11
	NSEventTypeFlagsChanged       = 12
	NSEventTypeApplicationDefined = 15
	NSEventTypeScrollWheel        = 22
)

// NSEvent masks.
const (
	NSEventMaskAny = 0xffffffffffffffff
)

// NSTrackingArea options.
const (
	NSTrackingMouseEnteredAndExited = 0x01
	NSTrackingActiveInKeyWindow     = 0x20
	NSTrackingCursorUpdate          = 0x04
	NSTrackingInVisibleRect         = 0x200
	NSTrackingAssumeInside          = 0x100
)

// NSWindow collection behaviors.
const (
	_NSWindowCollectionBehaviorFullScreenPrimary = 1 << 7
	_NSWindowCollectionBehaviorManaged           = 1 << 2
)

// Drag operations.
const (
	NSDragOperationGeneric = 4
)

// Occlusion state.
const (
	NSWindowOcclusionStateVisible = 1 << 1
)

// CoreGraphics / IOKit constants.
const (
	kCGDisplayModeIsInterlaced            = 0x00100000
	kCGDisplayModeIsStretched             = 0x00200000
	kCGDisplayFadeReservationInvalidToken = 0
	kIODisplayOnlyPreferredName           = 1
	kCGErrorSuccess                       = 0
)

// NSOpenGL pixel format attributes.
const (
	NSOpenGLPFAAllRenderers       = 1
	NSOpenGLPFADoubleBuffer       = 5
	NSOpenGLPFAStereo             = 6
	NSOpenGLPFAColorSize          = 8
	NSOpenGLPFAAlphaSize          = 11
	NSOpenGLPFADepthSize          = 12
	NSOpenGLPFAStencilSize        = 13
	NSOpenGLPFAAccumSize          = 14
	NSOpenGLPFASampleBuffers      = 55
	NSOpenGLPFASamples            = 56
	NSOpenGLPFAAuxBuffers         = 7
	NSOpenGLPFAClosestPolicy      = 74
	NSOpenGLPFAOpenGLProfile      = 99
	NSOpenGLPFAAccelerated        = 73
	NSOpenGLProfileVersion3_2Core = 0x3200
	NSOpenGLProfileVersion4_1Core = 0x4100
	NSOpenGLCPSwapInterval        = 222
	NSOpenGLCPSurfaceOpacity      = 236
)

// cgRect matches the CoreGraphics CGRect struct layout.
type cgRect struct {
	X, Y, Width, Height float64
}

// nsRange matches the Foundation NSRange struct layout.
type nsRange struct {
	Location uintptr
	Length   uintptr
}

// Framework handle for OpenGL.
var openGLFramework uintptr

// CoreFoundation function pointers.
var (
	cfBundleGetBundleWithIdentifier   func(bundleID uintptr) uintptr
	cfBundleGetFunctionPointerForName func(bundle uintptr, functionName uintptr) uintptr
	cfBundleGetDataPointerForName     func(bundle uintptr, symbolName uintptr) uintptr
	cfStringCreateWithCString         func(alloc uintptr, cStr string, encoding uint32) uintptr
	cfArrayGetCount                   func(array uintptr) int
	cfArrayGetValueAtIndex            func(array uintptr, index int) uintptr
	cfRelease                         func(cf uintptr)
)

// CoreGraphics function pointers.
var (
	cgEventSourceCreate                    func(stateID int32) uintptr
	cgMainDisplayID                        func() uint32
	cgWarpMouseCursorPosition              func(point cocoa.CGPoint) int32
	cgAssociateMouseAndMouseCursorPosition func(connected int32) int32
	cgDisplayMoveCursorToPoint             func(display uint32, point cocoa.CGPoint) int32
	cgGetOnlineDisplayList                 func(maxDisplays uint32, onlineDisplays *uint32, displayCount *uint32) int32
	cgDisplayBounds                        func(display uint32) cgRect
	cgDisplayCopyAllDisplayModes           func(display uint32, options uintptr) uintptr
	cgDisplayCopyDisplayMode               func(display uint32) uintptr
	cgDisplayModeGetWidth                  func(mode uintptr) uintptr
	cgDisplayModeGetHeight                 func(mode uintptr) uintptr
	cgDisplayModeGetRefreshRate            func(mode uintptr) float64
	cgDisplayModeGetPixelWidth             func(mode uintptr) uintptr
	cgDisplayModeGetPixelHeight            func(mode uintptr) uintptr
	cgDisplaySetDisplayMode                func(display uint32, mode uintptr, options uintptr) int32
	cgDisplayIsAsleep                      func(display uint32) uint32
	cgGetDisplayTransferByTable            func(display uint32, capacity uint32, red *float32, green *float32, blue *float32, sampleCount *uint32) int32
	cgSetDisplayTransferByTable            func(display uint32, sampleCount uint32, red *float32, green *float32, blue *float32) int32
	cgDisplayRestoreColorSyncSettings      func()
	cgAcquireDisplayFadeReservation        func(seconds float32, pNewToken *uint32) int32
	cgDisplayFade                          func(token uint32, duration float32, startColor float32, endColor float32, synchronous uint32) int32
	cgReleaseDisplayFadeReservation        func(token uint32) int32
	cgDisplayModelNumber                   func(display uint32) uint32
	cgDisplayVendorNumber                  func(display uint32) uint32
	cgDisplayUnitNumber                    func(display uint32) uint32
	cgDisplayModeGetIOFlags                func(mode uintptr) uint32
)

// IOKit function pointers.
var (
	ioServiceGetMatchingServices      func(mainPort uint32, matching uintptr, existing *uint32) int32
	ioIteratorNext                    func(iterator uint32) uint32
	ioRegistryEntryGetName            func(entry uint32, name *[128]byte) int32
	ioRegistryEntryCreateCFProperties func(entry uint32, properties *uintptr, allocator uintptr, options uint32) int32
	ioDisplayCreateInfoDictionary     func(framebuffer uint32, options uint32) uintptr
	ioServiceMatching                 func(name *byte) uintptr
	ioObjectRelease                   func(object uint32) int32
)

// ObjC classes (initialized in init after loading AppKit).
var (
	class_NSApplication        objc.Class
	class_NSMenu               objc.Class
	class_NSMenuItem           objc.Class
	class_NSEvent              objc.Class
	class_NSProcessInfo        objc.Class
	class_NSNotificationCenter objc.Class
	class_NSBundle             objc.Class
	class_NSScreen             objc.Class
	class_NSWindow             objc.Class
	class_NSView               objc.Class
	class_NSPasteboard         objc.Class
	class_NSCursor             objc.Class
	class_NSImage              objc.Class
	class_NSBitmapImageRep     objc.Class
	class_NSTrackingArea       objc.Class
	class_NSColor              objc.Class
	class_NSArray              objc.Class
	class_NSURL                objc.Class
	class_NSOpenGLPixelFormat  objc.Class
	class_NSOpenGLContext      objc.Class
)

// ObjC selectors.
var (
	// General
	sel_alloc   = objc.RegisterName("alloc")
	sel_init    = objc.RegisterName("init")
	sel_release = objc.RegisterName("release")
	sel_retain  = objc.RegisterName("retain")

	// NSApplication
	sel_NSApp                                 = objc.RegisterName("sharedApplication")
	sel_sharedApplication                     = objc.RegisterName("sharedApplication")
	sel_setActivationPolicy                   = objc.RegisterName("setActivationPolicy:")
	sel_setMainMenu                           = objc.RegisterName("setMainMenu:")
	sel_mainMenu                              = objc.RegisterName("mainMenu")
	sel_setWindowsMenu                        = objc.RegisterName("setWindowsMenu:")
	sel_setServicesMenu                       = objc.RegisterName("setServicesMenu:")
	sel_run                                   = objc.RegisterName("run")
	sel_stop                                  = objc.RegisterName("stop:")
	sel_nextEventMatchingMask                 = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	sel_sendEvent                             = objc.RegisterName("sendEvent:")
	sel_updateWindows                         = objc.RegisterName("updateWindows")
	sel_activateIgnoringOtherApps             = objc.RegisterName("activateIgnoringOtherApps:")
	sel_postEventAtStart                      = objc.RegisterName("postEvent:atStart:")
	sel_hide                                  = objc.RegisterName("hide:")
	sel_unhideAllApplications                 = objc.RegisterName("unhideAllApplications:")
	sel_hideOtherApplications                 = objc.RegisterName("hideOtherApplications:")
	sel_terminate                             = objc.RegisterName("terminate:")
	sel_orderFrontStandardAboutPanel          = objc.RegisterName("orderFrontStandardAboutPanel:")
	sel_addGlobalMonitorForEventsMatchingMask = objc.RegisterName("addGlobalMonitorForEventsMatchingMask:handler:")

	// NSProcessInfo
	sel_processInfo = objc.RegisterName("processInfo")
	sel_processName = objc.RegisterName("processName")

	// NSBundle
	sel_bundleIdentifier = objc.RegisterName("bundleIdentifier")
	sel_mainBundle       = objc.RegisterName("mainBundle")
	sel_infoDictionary   = objc.RegisterName("infoDictionary")

	// NSNotificationCenter
	sel_defaultCenter                 = objc.RegisterName("defaultCenter")
	sel_addObserverSelectorNameObject = objc.RegisterName("addObserver:selector:name:object:")

	// NSMenu / NSMenuItem
	sel_initWithTitle                = objc.RegisterName("initWithTitle:")
	sel_addItem                      = objc.RegisterName("addItem:")
	sel_addItemWithTitle             = objc.RegisterName("addItemWithTitle:action:keyEquivalent:")
	sel_setSubmenu                   = objc.RegisterName("setSubmenu:")
	sel_separatorItem                = objc.RegisterName("separatorItem")
	sel_setKeyEquivalentModifierMask = objc.RegisterName("setKeyEquivalentModifierMask:")

	// NSEvent type/modifier selectors
	sel_otherEventWithType          = objc.RegisterName("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:")
	sel_type                        = objc.RegisterName("type")
	sel_modifierFlags               = objc.RegisterName("modifierFlags")
	sel_keyCode                     = objc.RegisterName("keyCode")
	sel_characters                  = objc.RegisterName("characters")
	sel_charactersIgnoringModifiers = objc.RegisterName("charactersIgnoringModifiers")
	sel_locationInWindow            = objc.RegisterName("locationInWindow")
	sel_scrollingDeltaX             = objc.RegisterName("scrollingDeltaX")
	sel_scrollingDeltaY             = objc.RegisterName("scrollingDeltaY")
	sel_hasPreciseScrollingDeltas   = objc.RegisterName("hasPreciseScrollingDeltas")
	sel_buttonNumber                = objc.RegisterName("buttonNumber")
	sel_deltaX                      = objc.RegisterName("deltaX")
	sel_deltaY                      = objc.RegisterName("deltaY")

	// NSWindow selectors
	sel_setTitle                = objc.RegisterName("setTitle:")
	sel_setContentSize          = objc.RegisterName("setContentSize:")
	sel_setFrameOrigin          = objc.RegisterName("setFrameOrigin:")
	sel_makeKeyAndOrderFront    = objc.RegisterName("makeKeyAndOrderFront:")
	sel_orderOut                = objc.RegisterName("orderOut:")
	sel_miniaturize             = objc.RegisterName("miniaturize:")
	sel_deminiaturize           = objc.RegisterName("deminiaturize:")
	sel_zoom                    = objc.RegisterName("zoom:")
	sel_toggleFullScreen        = objc.RegisterName("toggleFullScreen:")
	sel_setOpaque               = objc.RegisterName("setOpaque:")
	sel_setHasShadow            = objc.RegisterName("setHasShadow:")
	sel_setBackgroundColor      = objc.RegisterName("setBackgroundColor:")
	sel_setLevel                = objc.RegisterName("setLevel:")
	sel_level                   = objc.RegisterName("level")
	sel_setContentView          = objc.RegisterName("setContentView:")
	sel_contentView             = objc.RegisterName("contentView")
	sel_setDelegate             = objc.RegisterName("setDelegate:")
	sel_delegate                = objc.RegisterName("delegate")
	sel_isKeyWindow             = objc.RegisterName("isKeyWindow")
	sel_isMiniaturized          = objc.RegisterName("isMiniaturized")
	sel_isVisible               = objc.RegisterName("isVisible")
	sel_isZoomed                = objc.RegisterName("isZoomed")
	sel_setMinSize              = objc.RegisterName("setMinSize:")
	sel_setMaxSize              = objc.RegisterName("setMaxSize:")
	sel_setContentAspectRatio   = objc.RegisterName("setContentAspectRatio:")
	sel_setRestorable           = objc.RegisterName("setRestorable:")
	sel_setCollectionBehavior   = objc.RegisterName("setCollectionBehavior:")
	sel_setIgnoresMouseEvents   = objc.RegisterName("setIgnoresMouseEvents:")
	sel_alphaValue              = objc.RegisterName("alphaValue")
	sel_setAlphaValue           = objc.RegisterName("setAlphaValue:")
	sel_occlusionState          = objc.RegisterName("occlusionState")
	sel_windowNumber            = objc.RegisterName("windowNumber")
	sel_convertRectToBacking    = objc.RegisterName("convertRectToBacking:")
	sel_convertRectFromBacking  = objc.RegisterName("convertRectFromBacking:")
	sel_convertPointToBacking   = objc.RegisterName("convertPointToBacking:")
	sel_convertPointFromBacking = objc.RegisterName("convertPointFromBacking:")
	sel_initWithContentRect     = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	sel_contentRectForFrameRect = objc.RegisterName("contentRectForFrameRect:")
	sel_frameRectForContentRect = objc.RegisterName("frameRectForContentRect:")
	sel_requestUserAttention    = objc.RegisterName("requestUserAttention:")
	sel_arrangeInFront          = objc.RegisterName("arrangeInFront:")

	// NSView selectors
	sel_frame              = objc.RegisterName("frame")
	sel_bounds             = objc.RegisterName("bounds")
	sel_window             = objc.RegisterName("window")
	sel_addTrackingArea    = objc.RegisterName("addTrackingArea:")
	sel_removeTrackingArea = objc.RegisterName("removeTrackingArea:")
	sel_trackingAreas      = objc.RegisterName("trackingAreas")
	sel_setNeedsDisplay    = objc.RegisterName("setNeedsDisplay:")

	// NSScreen selectors
	sel_screens            = objc.RegisterName("screens")
	sel_mainScreen         = objc.RegisterName("mainScreen")
	sel_screen             = objc.RegisterName("screen")
	sel_deviceDescription  = objc.RegisterName("deviceDescription")
	sel_backingScaleFactor = objc.RegisterName("backingScaleFactor")
	sel_visibleFrame       = objc.RegisterName("visibleFrame")

	// NSPasteboard selectors
	sel_generalPasteboard = objc.RegisterName("generalPasteboard")
	sel_declareTypes      = objc.RegisterName("declareTypes:owner:")
	sel_setStringForType  = objc.RegisterName("setString:forType:")
	sel_stringForType     = objc.RegisterName("stringForType:")

	// NSCursor selectors
	sel_arrowCursor               = objc.RegisterName("arrowCursor")
	sel_IBeamCursor               = objc.RegisterName("IBeamCursor")
	sel_crosshairCursor           = objc.RegisterName("crosshairCursor")
	sel_closedHandCursor          = objc.RegisterName("closedHandCursor")
	sel_openHandCursor            = objc.RegisterName("openHandCursor")
	sel_pointingHandCursor        = objc.RegisterName("pointingHandCursor")
	sel_resizeLeftCursor          = objc.RegisterName("resizeLeftCursor")
	sel_resizeRightCursor         = objc.RegisterName("resizeRightCursor")
	sel_resizeLeftRightCursor     = objc.RegisterName("resizeLeftRightCursor")
	sel_resizeUpCursor            = objc.RegisterName("resizeUpCursor")
	sel_resizeDownCursor          = objc.RegisterName("resizeDownCursor")
	sel_resizeUpDownCursor        = objc.RegisterName("resizeUpDownCursor")
	sel_operationNotAllowedCursor = objc.RegisterName("operationNotAllowedCursor")
	sel_set_cursor                = objc.RegisterName("set")
	sel_hide_cursor               = objc.RegisterName("hide")
	sel_unhide_cursor             = objc.RegisterName("unhide")

	// NSImage / NSBitmapImageRep selectors
	sel_initWithSize             = objc.RegisterName("initWithSize:")
	sel_addRepresentation        = objc.RegisterName("addRepresentation:")
	sel_initWithBitmapDataPlanes = objc.RegisterName("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bitmapFormat:bytesPerRow:bitsPerPixel:")
	sel_bitmapData               = objc.RegisterName("bitmapData")

	// NSTrackingArea selectors
	sel_initWithRect_options_owner_userInfo = objc.RegisterName("initWithRect:options:owner:userInfo:")

	// NSColor selectors
	sel_clearColor = objc.RegisterName("clearColor")

	// NSArray selectors
	sel_arrayWithObject  = objc.RegisterName("arrayWithObject:")
	sel_count            = objc.RegisterName("count")
	sel_objectAtIndex    = objc.RegisterName("objectAtIndex:")
	sel_objectForKey     = objc.RegisterName("objectForKey:")
	sel_unsignedIntValue = objc.RegisterName("unsignedIntValue")
	sel_localizedName    = objc.RegisterName("localizedName")
	sel_UTF8String       = objc.RegisterName("UTF8String")
	sel_length           = objc.RegisterName("length")

	// NSURL selectors
	sel_path = objc.RegisterName("path")

	// Drag and drop selectors
	sel_draggingPasteboard      = objc.RegisterName("draggingPasteboard")
	sel_readObjectsForClasses   = objc.RegisterName("readObjectsForClasses:options:")
	sel_registerForDraggedTypes = objc.RegisterName("registerForDraggedTypes:")

	// Text input selectors
	sel_interpretKeyEvents                  = objc.RegisterName("interpretKeyEvents:")
	sel_hasMarkedText                       = objc.RegisterName("hasMarkedText")
	sel_markedRange                         = objc.RegisterName("markedRange")
	sel_selectedRange                       = objc.RegisterName("selectedRange")
	sel_setMarkedText                       = objc.RegisterName("setMarkedText:selectedRange:replacementRange:")
	sel_unmarkText                          = objc.RegisterName("unmarkText")
	sel_validAttributesForMarkedText        = objc.RegisterName("validAttributesForMarkedText")
	sel_attributedSubstringForProposedRange = objc.RegisterName("attributedSubstringForProposedRange:actualRange:")
	sel_insertText                          = objc.RegisterName("insertText:replacementRange:")
	sel_characterIndexForPoint              = objc.RegisterName("characterIndexForPoint:")
	sel_firstRectForCharacterRange          = objc.RegisterName("firstRectForCharacterRange:actualRange:")
	sel_doCommandBySelector                 = objc.RegisterName("doCommandBySelector:")

	// GLFWApplicationDelegate selectors
	sel_selectedKeyboardInputSourceChanged   = objc.RegisterName("selectedKeyboardInputSourceChanged:")
	sel_applicationShouldTerminate           = objc.RegisterName("applicationShouldTerminate:")
	sel_applicationDidChangeScreenParameters = objc.RegisterName("applicationDidChangeScreenParameters:")
	sel_applicationWillFinishLaunching       = objc.RegisterName("applicationWillFinishLaunching:")
	sel_applicationDidFinishLaunching        = objc.RegisterName("applicationDidFinishLaunching:")
	sel_applicationDidHide                   = objc.RegisterName("applicationDidHide:")

	// NSOpenGL selectors
	sel_initWithAttributes                  = objc.RegisterName("initWithAttributes:")
	sel_initWithFormat_shareContext         = objc.RegisterName("initWithFormat:shareContext:")
	sel_makeCurrentContext                  = objc.RegisterName("makeCurrentContext")
	sel_clearCurrentContext                 = objc.RegisterName("clearCurrentContext")
	sel_flushBuffer                         = objc.RegisterName("flushBuffer")
	sel_setValues_forParameter              = objc.RegisterName("setValues:forParameter:")
	sel_getValues_forParameter              = objc.RegisterName("getValues:forParameter:")
	sel_setView                             = objc.RegisterName("setView:")
	sel_clearDrawable                       = objc.RegisterName("clearDrawable")
	sel_setWantsBestResolutionOpenGLSurface = objc.RegisterName("setWantsBestResolutionOpenGLSurface:")
)

func init() {
	// Load CoreFoundation.
	coreFoundation, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen CoreFoundation: %w", err))
	}
	purego.RegisterLibFunc(&cfBundleGetBundleWithIdentifier, coreFoundation, "CFBundleGetBundleWithIdentifier")
	purego.RegisterLibFunc(&cfBundleGetFunctionPointerForName, coreFoundation, "CFBundleGetFunctionPointerForName")
	purego.RegisterLibFunc(&cfBundleGetDataPointerForName, coreFoundation, "CFBundleGetDataPointerForName")
	purego.RegisterLibFunc(&cfStringCreateWithCString, coreFoundation, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&cfArrayGetCount, coreFoundation, "CFArrayGetCount")
	purego.RegisterLibFunc(&cfArrayGetValueAtIndex, coreFoundation, "CFArrayGetValueAtIndex")
	purego.RegisterLibFunc(&cfRelease, coreFoundation, "CFRelease")

	// Load CoreGraphics.
	coreGraphics, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen CoreGraphics: %w", err))
	}
	purego.RegisterLibFunc(&cgEventSourceCreate, coreGraphics, "CGEventSourceCreate")
	purego.RegisterLibFunc(&cgMainDisplayID, coreGraphics, "CGMainDisplayID")
	purego.RegisterLibFunc(&cgWarpMouseCursorPosition, coreGraphics, "CGWarpMouseCursorPosition")
	purego.RegisterLibFunc(&cgAssociateMouseAndMouseCursorPosition, coreGraphics, "CGAssociateMouseAndMouseCursorPosition")
	purego.RegisterLibFunc(&cgDisplayMoveCursorToPoint, coreGraphics, "CGDisplayMoveCursorToPoint")
	purego.RegisterLibFunc(&cgGetOnlineDisplayList, coreGraphics, "CGGetOnlineDisplayList")
	purego.RegisterLibFunc(&cgDisplayBounds, coreGraphics, "CGDisplayBounds")
	purego.RegisterLibFunc(&cgDisplayCopyAllDisplayModes, coreGraphics, "CGDisplayCopyAllDisplayModes")
	purego.RegisterLibFunc(&cgDisplayCopyDisplayMode, coreGraphics, "CGDisplayCopyDisplayMode")
	purego.RegisterLibFunc(&cgDisplayModeGetWidth, coreGraphics, "CGDisplayModeGetWidth")
	purego.RegisterLibFunc(&cgDisplayModeGetHeight, coreGraphics, "CGDisplayModeGetHeight")
	purego.RegisterLibFunc(&cgDisplayModeGetRefreshRate, coreGraphics, "CGDisplayModeGetRefreshRate")
	purego.RegisterLibFunc(&cgDisplayModeGetPixelWidth, coreGraphics, "CGDisplayModeGetPixelWidth")
	purego.RegisterLibFunc(&cgDisplayModeGetPixelHeight, coreGraphics, "CGDisplayModeGetPixelHeight")
	purego.RegisterLibFunc(&cgDisplaySetDisplayMode, coreGraphics, "CGDisplaySetDisplayMode")
	purego.RegisterLibFunc(&cgDisplayIsAsleep, coreGraphics, "CGDisplayIsAsleep")
	purego.RegisterLibFunc(&cgGetDisplayTransferByTable, coreGraphics, "CGGetDisplayTransferByTable")
	purego.RegisterLibFunc(&cgSetDisplayTransferByTable, coreGraphics, "CGSetDisplayTransferByTable")
	purego.RegisterLibFunc(&cgDisplayRestoreColorSyncSettings, coreGraphics, "CGDisplayRestoreColorSyncSettings")
	purego.RegisterLibFunc(&cgAcquireDisplayFadeReservation, coreGraphics, "CGAcquireDisplayFadeReservation")
	purego.RegisterLibFunc(&cgDisplayFade, coreGraphics, "CGDisplayFade")
	purego.RegisterLibFunc(&cgReleaseDisplayFadeReservation, coreGraphics, "CGReleaseDisplayFadeReservation")
	purego.RegisterLibFunc(&cgDisplayModelNumber, coreGraphics, "CGDisplayModelNumber")
	purego.RegisterLibFunc(&cgDisplayVendorNumber, coreGraphics, "CGDisplayVendorNumber")
	purego.RegisterLibFunc(&cgDisplayUnitNumber, coreGraphics, "CGDisplayUnitNumber")
	purego.RegisterLibFunc(&cgDisplayModeGetIOFlags, coreGraphics, "CGDisplayModeGetIOFlags")

	// Load IOKit.
	ioKit, err := purego.Dlopen("/System/Library/Frameworks/IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen IOKit: %w", err))
	}
	purego.RegisterLibFunc(&ioServiceGetMatchingServices, ioKit, "IOServiceGetMatchingServices")
	purego.RegisterLibFunc(&ioIteratorNext, ioKit, "IOIteratorNext")
	purego.RegisterLibFunc(&ioRegistryEntryGetName, ioKit, "IORegistryEntryGetName")
	purego.RegisterLibFunc(&ioRegistryEntryCreateCFProperties, ioKit, "IORegistryEntryCreateCFProperties")
	purego.RegisterLibFunc(&ioDisplayCreateInfoDictionary, ioKit, "IODisplayCreateInfoDictionary")
	purego.RegisterLibFunc(&ioServiceMatching, ioKit, "IOServiceMatching")
	purego.RegisterLibFunc(&ioObjectRelease, ioKit, "IOObjectRelease")

	// Load AppKit (required for NSApplication, NSWindow, NSCursor, etc.).
	_, err = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen AppKit: %w", err))
	}

	// Load OpenGL.
	openGLFramework, err = purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/OpenGL", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen OpenGL: %w", err))
	}

	// Look up ObjC classes (must be after loading AppKit).
	class_NSApplication = objc.GetClass("NSApplication")
	class_NSMenu = objc.GetClass("NSMenu")
	class_NSMenuItem = objc.GetClass("NSMenuItem")
	class_NSEvent = objc.GetClass("NSEvent")
	class_NSProcessInfo = objc.GetClass("NSProcessInfo")
	class_NSNotificationCenter = objc.GetClass("NSNotificationCenter")
	class_NSBundle = objc.GetClass("NSBundle")
	class_NSScreen = objc.GetClass("NSScreen")
	class_NSWindow = objc.GetClass("NSWindow")
	class_NSView = objc.GetClass("NSView")
	class_NSPasteboard = objc.GetClass("NSPasteboard")
	class_NSCursor = objc.GetClass("NSCursor")
	class_NSImage = objc.GetClass("NSImage")
	class_NSBitmapImageRep = objc.GetClass("NSBitmapImageRep")
	class_NSTrackingArea = objc.GetClass("NSTrackingArea")
	class_NSColor = objc.GetClass("NSColor")
	class_NSArray = objc.GetClass("NSArray")
	class_NSURL = objc.GetClass("NSURL")
	class_NSOpenGLPixelFormat = objc.GetClass("NSOpenGLPixelFormat")
	class_NSOpenGLContext = objc.GetClass("NSOpenGLContext")
}

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
	_NSWindowCollectionBehaviorFullScreenNone    = 1 << 9
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
	classNSApplication        objc.Class
	classNSMenu               objc.Class
	classNSMenuItem           objc.Class
	classNSEvent              objc.Class
	classNSProcessInfo        objc.Class
	classNSNotificationCenter objc.Class
	classNSBundle             objc.Class
	classNSScreen             objc.Class
	classNSWindow             objc.Class
	classNSView               objc.Class
	classNSPasteboard         objc.Class
	classNSCursor             objc.Class
	classNSImage              objc.Class
	classNSBitmapImageRep     objc.Class
	classNSTrackingArea       objc.Class
	classNSColor              objc.Class
	classNSArray              objc.Class
	classNSURL                objc.Class
	classNSOpenGLPixelFormat  objc.Class
	classNSOpenGLContext      objc.Class
)

// ObjC selectors.
var (
	// General
	selAlloc   = objc.RegisterName("alloc")
	selInit    = objc.RegisterName("init")
	selRelease = objc.RegisterName("release")
	selRetain  = objc.RegisterName("retain")

	// NSApplication
	selNSApp                                 = objc.RegisterName("sharedApplication")
	selSharedApplication                     = objc.RegisterName("sharedApplication")
	selSetActivationPolicy                   = objc.RegisterName("setActivationPolicy:")
	selSetMainMenu                           = objc.RegisterName("setMainMenu:")
	selMainMenu                              = objc.RegisterName("mainMenu")
	selSetWindowsMenu                        = objc.RegisterName("setWindowsMenu:")
	selSetServicesMenu                       = objc.RegisterName("setServicesMenu:")
	selRun                                   = objc.RegisterName("run")
	selStop                                  = objc.RegisterName("stop:")
	selNextEventMatchingMask                 = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent                             = objc.RegisterName("sendEvent:")
	selUpdateWindows                         = objc.RegisterName("updateWindows")
	selActivateIgnoringOtherApps             = objc.RegisterName("activateIgnoringOtherApps:")
	selPostEventAtStart                      = objc.RegisterName("postEvent:atStart:")
	selHide                                  = objc.RegisterName("hide:")
	selUnhideAllApplications                 = objc.RegisterName("unhideAllApplications:")
	selHideOtherApplications                 = objc.RegisterName("hideOtherApplications:")
	selTerminate                             = objc.RegisterName("terminate:")
	selOrderFrontStandardAboutPanel          = objc.RegisterName("orderFrontStandardAboutPanel:")
	selAddGlobalMonitorForEventsMatchingMask = objc.RegisterName("addGlobalMonitorForEventsMatchingMask:handler:")

	// NSProcessInfo
	selProcessInfo = objc.RegisterName("processInfo")
	selProcessName = objc.RegisterName("processName")

	// NSBundle
	selBundleIdentifier = objc.RegisterName("bundleIdentifier")
	selMainBundle       = objc.RegisterName("mainBundle")
	selInfoDictionary   = objc.RegisterName("infoDictionary")

	// NSNotificationCenter
	selDefaultCenter                 = objc.RegisterName("defaultCenter")
	selAddObserverSelectorNameObject = objc.RegisterName("addObserver:selector:name:object:")

	// NSMenu / NSMenuItem
	selInitWithTitle                = objc.RegisterName("initWithTitle:")
	selAddItem                      = objc.RegisterName("addItem:")
	selAddItemWithTitle             = objc.RegisterName("addItemWithTitle:action:keyEquivalent:")
	selSetSubmenu                   = objc.RegisterName("setSubmenu:")
	selSeparatorItem                = objc.RegisterName("separatorItem")
	selSetKeyEquivalentModifierMask = objc.RegisterName("setKeyEquivalentModifierMask:")

	// NSEvent type/modifier selectors
	selCurrentEvent                = objc.RegisterName("currentEvent")
	selOtherEventWithType          = objc.RegisterName("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:")
	selType                        = objc.RegisterName("type")
	selModifierFlags               = objc.RegisterName("modifierFlags")
	selKeyCode                     = objc.RegisterName("keyCode")
	selCharacters                  = objc.RegisterName("characters")
	selCharactersIgnoringModifiers = objc.RegisterName("charactersIgnoringModifiers")
	selLocationInWindow            = objc.RegisterName("locationInWindow")
	selScrollingDeltaX             = objc.RegisterName("scrollingDeltaX")
	selScrollingDeltaY             = objc.RegisterName("scrollingDeltaY")
	selHasPreciseScrollingDeltas   = objc.RegisterName("hasPreciseScrollingDeltas")
	selButtonNumber                = objc.RegisterName("buttonNumber")
	selDeltaX                      = objc.RegisterName("deltaX")
	selDeltaY                      = objc.RegisterName("deltaY")

	// NSWindow selectors
	selSetTitle                          = objc.RegisterName("setTitle:")
	selSetContentSize                    = objc.RegisterName("setContentSize:")
	selSetFrameOrigin                    = objc.RegisterName("setFrameOrigin:")
	selMakeKeyAndOrderFront              = objc.RegisterName("makeKeyAndOrderFront:")
	selOrderOut                          = objc.RegisterName("orderOut:")
	selMiniaturize                       = objc.RegisterName("miniaturize:")
	selDeminiaturize                     = objc.RegisterName("deminiaturize:")
	selZoom                              = objc.RegisterName("zoom:")
	selToggleFullScreen                  = objc.RegisterName("toggleFullScreen:")
	selSetOpaque                         = objc.RegisterName("setOpaque:")
	selSetHasShadow                      = objc.RegisterName("setHasShadow:")
	selSetBackgroundColor                = objc.RegisterName("setBackgroundColor:")
	selSetLevel                          = objc.RegisterName("setLevel:")
	selLevel                             = objc.RegisterName("level")
	selSetContentView                    = objc.RegisterName("setContentView:")
	selContentView                       = objc.RegisterName("contentView")
	selSetDelegate                       = objc.RegisterName("setDelegate:")
	selDelegate                          = objc.RegisterName("delegate")
	selIsKeyWindow                       = objc.RegisterName("isKeyWindow")
	selIsMiniaturized                    = objc.RegisterName("isMiniaturized")
	selIsVisible                         = objc.RegisterName("isVisible")
	selIsZoomed                          = objc.RegisterName("isZoomed")
	selSetMinSize                        = objc.RegisterName("setMinSize:")
	selSetMaxSize                        = objc.RegisterName("setMaxSize:")
	selSetContentMinSize                 = objc.RegisterName("setContentMinSize:")
	selSetContentMaxSize                 = objc.RegisterName("setContentMaxSize:")
	selSetContentAspectRatio             = objc.RegisterName("setContentAspectRatio:")
	selSetResizeIncrements               = objc.RegisterName("setResizeIncrements:")
	selOrderFront                        = objc.RegisterName("orderFront:")
	selStyleMask                         = objc.RegisterName("styleMask")
	selSetStyleMask                      = objc.RegisterName("setStyleMask:")
	selMiniwindowTitle                   = objc.RegisterName("miniwindowTitle")
	selSetMiniwindowTitle                = objc.RegisterName("setMiniwindowTitle:")
	selMakeFirstResponder                = objc.RegisterName("makeFirstResponder:")
	selSetRestorable                     = objc.RegisterName("setRestorable:")
	selSetCollectionBehavior             = objc.RegisterName("setCollectionBehavior:")
	selSetIgnoresMouseEvents             = objc.RegisterName("setIgnoresMouseEvents:")
	selAlphaValue                        = objc.RegisterName("alphaValue")
	selSetAlphaValue                     = objc.RegisterName("setAlphaValue:")
	selOcclusionState                    = objc.RegisterName("occlusionState")
	selWindowNumber                      = objc.RegisterName("windowNumber")
	selConvertRectToBacking              = objc.RegisterName("convertRectToBacking:")
	selConvertRectFromBacking            = objc.RegisterName("convertRectFromBacking:")
	selConvertPointToBacking             = objc.RegisterName("convertPointToBacking:")
	selConvertPointFromBacking           = objc.RegisterName("convertPointFromBacking:")
	selInitWithContentRect               = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	selContentRectForFrameRect           = objc.RegisterName("contentRectForFrameRect:")
	selFrameRectForContentRect           = objc.RegisterName("frameRectForContentRect:")
	selRequestUserAttention              = objc.RegisterName("requestUserAttention:")
	selArrangeInFront                    = objc.RegisterName("arrangeInFront:")
	selConvertRectToScreen               = objc.RegisterName("convertRectToScreen:")
	selMouseLocationOutsideOfEventStream = objc.RegisterName("mouseLocationOutsideOfEventStream")

	// NSView selectors
	selFrame              = objc.RegisterName("frame")
	selBounds             = objc.RegisterName("bounds")
	selWindow             = objc.RegisterName("window")
	selAddTrackingArea    = objc.RegisterName("addTrackingArea:")
	selRemoveTrackingArea = objc.RegisterName("removeTrackingArea:")
	selTrackingAreas      = objc.RegisterName("trackingAreas")
	selSetNeedsDisplay    = objc.RegisterName("setNeedsDisplay:")

	// NSScreen selectors
	selScreens            = objc.RegisterName("screens")
	selMainScreen         = objc.RegisterName("mainScreen")
	selScreen             = objc.RegisterName("screen")
	selDeviceDescription  = objc.RegisterName("deviceDescription")
	selBackingScaleFactor = objc.RegisterName("backingScaleFactor")
	selVisibleFrame       = objc.RegisterName("visibleFrame")

	// NSPasteboard selectors
	selGeneralPasteboard = objc.RegisterName("generalPasteboard")
	selDeclareTypes      = objc.RegisterName("declareTypes:owner:")
	selSetStringForType  = objc.RegisterName("setString:forType:")
	selStringForType     = objc.RegisterName("stringForType:")

	// NSCursor selectors
	selArrowCursor               = objc.RegisterName("arrowCursor")
	selIBeamCursor               = objc.RegisterName("IBeamCursor")
	selCrosshairCursor           = objc.RegisterName("crosshairCursor")
	selClosedHandCursor          = objc.RegisterName("closedHandCursor")
	selOpenHandCursor            = objc.RegisterName("openHandCursor")
	selPointingHandCursor        = objc.RegisterName("pointingHandCursor")
	selResizeLeftCursor          = objc.RegisterName("resizeLeftCursor")
	selResizeRightCursor         = objc.RegisterName("resizeRightCursor")
	selResizeLeftRightCursor     = objc.RegisterName("resizeLeftRightCursor")
	selResizeUpCursor            = objc.RegisterName("resizeUpCursor")
	selResizeDownCursor          = objc.RegisterName("resizeDownCursor")
	selResizeUpDownCursor        = objc.RegisterName("resizeUpDownCursor")
	selOperationNotAllowedCursor = objc.RegisterName("operationNotAllowedCursor")
	selSetCursor                 = objc.RegisterName("set")
	selHideCursor                = objc.RegisterName("hide")
	selUnhideCursor              = objc.RegisterName("unhide")

	// NSImage / NSBitmapImageRep selectors
	selInitWithSize             = objc.RegisterName("initWithSize:")
	selAddRepresentation        = objc.RegisterName("addRepresentation:")
	selInitWithBitmapDataPlanes = objc.RegisterName("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bitmapFormat:bytesPerRow:bitsPerPixel:")
	selBitmapData               = objc.RegisterName("bitmapData")

	// NSTrackingArea selectors
	selInitWithRectOptionsOwnerUserInfo = objc.RegisterName("initWithRect:options:owner:userInfo:")

	// NSColor selectors
	selClearColor = objc.RegisterName("clearColor")

	// NSArray selectors
	selArrayWithObject  = objc.RegisterName("arrayWithObject:")
	selCount            = objc.RegisterName("count")
	selObjectAtIndex    = objc.RegisterName("objectAtIndex:")
	selObjectForKey     = objc.RegisterName("objectForKey:")
	selUnsignedIntValue = objc.RegisterName("unsignedIntValue")
	selLocalizedName    = objc.RegisterName("localizedName")
	selUTF8String       = objc.RegisterName("UTF8String")
	selLength           = objc.RegisterName("length")

	// NSURL selectors
	selPath = objc.RegisterName("path")

	// Drag and drop selectors
	selDraggingPasteboard      = objc.RegisterName("draggingPasteboard")
	selReadObjectsForClasses   = objc.RegisterName("readObjectsForClasses:options:")
	selRegisterForDraggedTypes = objc.RegisterName("registerForDraggedTypes:")

	// Text input selectors
	selInterpretKeyEvents                  = objc.RegisterName("interpretKeyEvents:")
	selHasMarkedText                       = objc.RegisterName("hasMarkedText")
	selMarkedRange                         = objc.RegisterName("markedRange")
	selSelectedRange                       = objc.RegisterName("selectedRange")
	selSetMarkedText                       = objc.RegisterName("setMarkedText:selectedRange:replacementRange:")
	selUnmarkText                          = objc.RegisterName("unmarkText")
	selValidAttributesForMarkedText        = objc.RegisterName("validAttributesForMarkedText")
	selAttributedSubstringForProposedRange = objc.RegisterName("attributedSubstringForProposedRange:actualRange:")
	selInsertText                          = objc.RegisterName("insertText:replacementRange:")
	selCharacterIndexForPoint              = objc.RegisterName("characterIndexForPoint:")
	selFirstRectForCharacterRange          = objc.RegisterName("firstRectForCharacterRange:actualRange:")
	selDoCommandBySelector                 = objc.RegisterName("doCommandBySelector:")

	// GLFWApplicationDelegate selectors
	selSelectedKeyboardInputSourceChanged   = objc.RegisterName("selectedKeyboardInputSourceChanged:")
	selApplicationShouldTerminate           = objc.RegisterName("applicationShouldTerminate:")
	selApplicationDidChangeScreenParameters = objc.RegisterName("applicationDidChangeScreenParameters:")
	selApplicationWillFinishLaunching       = objc.RegisterName("applicationWillFinishLaunching:")
	selApplicationDidFinishLaunching        = objc.RegisterName("applicationDidFinishLaunching:")
	selApplicationDidHide                   = objc.RegisterName("applicationDidHide:")

	// NSOpenGL selectors
	selInitWithAttributes                  = objc.RegisterName("initWithAttributes:")
	selInitWithFormatShareContext          = objc.RegisterName("initWithFormat:shareContext:")
	selMakeCurrentContext                  = objc.RegisterName("makeCurrentContext")
	selClearCurrentContext                 = objc.RegisterName("clearCurrentContext")
	selFlushBuffer                         = objc.RegisterName("flushBuffer")
	selSetValuesForParameter               = objc.RegisterName("setValues:forParameter:")
	selGetValuesForParameter               = objc.RegisterName("getValues:forParameter:")
	selSetView                             = objc.RegisterName("setView:")
	selClearDrawable                       = objc.RegisterName("clearDrawable")
	selSetWantsBestResolutionOpenGLSurface = objc.RegisterName("setWantsBestResolutionOpenGLSurface:")
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
	classNSApplication = objc.GetClass("NSApplication")
	classNSMenu = objc.GetClass("NSMenu")
	classNSMenuItem = objc.GetClass("NSMenuItem")
	classNSEvent = objc.GetClass("NSEvent")
	classNSProcessInfo = objc.GetClass("NSProcessInfo")
	classNSNotificationCenter = objc.GetClass("NSNotificationCenter")
	classNSBundle = objc.GetClass("NSBundle")
	classNSScreen = objc.GetClass("NSScreen")
	classNSWindow = objc.GetClass("NSWindow")
	classNSView = objc.GetClass("NSView")
	classNSPasteboard = objc.GetClass("NSPasteboard")
	classNSCursor = objc.GetClass("NSCursor")
	classNSImage = objc.GetClass("NSImage")
	classNSBitmapImageRep = objc.GetClass("NSBitmapImageRep")
	classNSTrackingArea = objc.GetClass("NSTrackingArea")
	classNSColor = objc.GetClass("NSColor")
	classNSArray = objc.GetClass("NSArray")
	classNSURL = objc.GetClass("NSURL")
	classNSOpenGLPixelFormat = objc.GetClass("NSOpenGLPixelFormat")
	classNSOpenGLContext = objc.GetClass("NSOpenGLContext")
}

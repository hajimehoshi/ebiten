package gamepad

import (
	"github.com/hajimehoshi/ebiten/v2/internal/corefoundation"
	"unsafe"

	"github.com/ebitengine/purego"
)

const kIOReturnSuccess = 0

const kIOHIDOptionsTypeNone _IOOptionBits = 0

const (
	kIOHIDElementTypeInput_Misc   = 1
	kIOHIDElementTypeInput_Button = 2
	kIOHIDElementTypeInput_Axis   = 3
)

const (
	kHIDPage_GenericDesktop = 0x1
	kHIDPage_Simulation     = 0x2
	kHIDPage_Button         = 0x9
	kHIDPage_Consumer       = 0x0C
)

const (
	kHIDUsage_GD_Joystick            = 0x4
	kHIDUsage_GD_GamePad             = 0x5
	kHIDUsage_GD_MultiAxisController = 0x8
	kHIDUsage_GD_X                   = 0x30
	kHIDUsage_GD_Y                   = 0x31
	kHIDUsage_GD_Z                   = 0x32
	kHIDUsage_GD_Rx                  = 0x33
	kHIDUsage_GD_Ry                  = 0x34
	kHIDUsage_GD_Rz                  = 0x35
	kHIDUsage_GD_Slider              = 0x36
	kHIDUsage_GD_Dial                = 0x37
	kHIDUsage_GD_Wheel               = 0x38
	kHIDUsage_GD_Hatswitch           = 0x39
	kHIDUsage_GD_Start               = 0x3D
	kHIDUsage_GD_Select              = 0x3E
	kHIDUsage_GD_SystemMainMenu      = 0x85
	kHIDUsage_GD_DPadUp              = 0x90
	kHIDUsage_GD_DPadDown            = 0x91
	kHIDUsage_GD_DPadRight           = 0x92
	kHIDUsage_GD_DPadLeft            = 0x93
	kHIDUsage_Sim_Rudder             = 0xBA
	kHIDUsage_Sim_Throttle           = 0xBB
	kHIDUsage_Sim_Accelerator        = 0xC4
	kHIDUsage_Sim_Brake              = 0xC5
	kHIDUsage_Sim_Steering           = 0xC8
)

var (
	kIOHIDVendorIDKey        = []byte("VendorID\x00")
	kIOHIDProductIDKey       = []byte("ProductID\x00")
	kIOHIDVersionNumberKey   = []byte("VersionNumber\x00")
	kIOHIDProductKey         = []byte("Product\x00")
	kIOHIDDeviceUsagePageKey = []byte("DeviceUsagePage\x00")
	kIOHIDDeviceUsageKey     = []byte("DeviceUsage\x00")
)

type _IOOptionBits uint32
type _IOHIDManagerRef uintptr
type _IOHIDDeviceRef uintptr
type _IOHIDElementRef uintptr
type _IOHIDValueRef uintptr
type _IOReturn int
type _IOHIDElementType uint

type _IOHIDDeviceCallback func(context unsafe.Pointer, result _IOReturn, sender unsafe.Pointer, device _IOHIDDeviceRef)

func init() {
	iokit, _ := purego.Dlopen("IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)

	purego.RegisterLibFunc(&_IOHIDElementGetTypeID, iokit, "IOHIDElementGetTypeID")
	purego.RegisterLibFunc(&_IOHIDManagerCreate, iokit, "IOHIDManagerCreate")
	purego.RegisterLibFunc(&_IOHIDDeviceGetProperty, iokit, "IOHIDDeviceGetProperty")
	purego.RegisterLibFunc(&_IOHIDManagerOpen, iokit, "IOHIDManagerOpen")
	purego.RegisterLibFunc(&_IOHIDManagerSetDeviceMatchingMultiple, iokit, "IOHIDManagerSetDeviceMatchingMultiple")
	purego.RegisterLibFunc(&_IOHIDManagerRegisterDeviceMatchingCallback, iokit, "IOHIDManagerRegisterDeviceMatchingCallback")
	purego.RegisterLibFunc(&_IOHIDManagerRegisterDeviceRemovalCallback, iokit, "IOHIDManagerRegisterDeviceRemovalCallback")
	purego.RegisterLibFunc(&_IOHIDManagerScheduleWithRunLoop, iokit, "IOHIDManagerScheduleWithRunLoop")
	purego.RegisterLibFunc(&_IOHIDElementGetType, iokit, "IOHIDElementGetType")
	purego.RegisterLibFunc(&_IOHIDElementGetUsage, iokit, "IOHIDElementGetUsage")
	purego.RegisterLibFunc(&_IOHIDElementGetUsagePage, iokit, "IOHIDElementGetUsagePage")
	purego.RegisterLibFunc(&_IOHIDElementGetLogicalMin, iokit, "IOHIDElementGetLogicalMin")
	purego.RegisterLibFunc(&_IOHIDElementGetLogicalMax, iokit, "IOHIDElementGetLogicalMax")
	purego.RegisterLibFunc(&_IOHIDDeviceGetValue, iokit, "IOHIDDeviceGetValue")
	purego.RegisterLibFunc(&_IOHIDValueGetIntegerValue, iokit, "IOHIDValueGetIntegerValue")
	purego.RegisterLibFunc(&_IOHIDDeviceCopyMatchingElements, iokit, "IOHIDDeviceCopyMatchingElements")
}

var _IOHIDElementGetTypeID func() corefoundation.CFTypeID

var _IOHIDManagerCreate func(allocator corefoundation.CFAllocatorRef, options _IOOptionBits) _IOHIDManagerRef

var _IOHIDDeviceGetProperty func(device _IOHIDDeviceRef, key corefoundation.CFStringRef) corefoundation.CFTypeRef

var _IOHIDManagerOpen func(manager _IOHIDManagerRef, options _IOOptionBits) _IOReturn

var _IOHIDManagerSetDeviceMatchingMultiple func(manager _IOHIDManagerRef, multiple corefoundation.CFArrayRef)

var _IOHIDManagerRegisterDeviceMatchingCallback func(manager _IOHIDManagerRef, callback _IOHIDDeviceCallback, context unsafe.Pointer)

var _IOHIDManagerRegisterDeviceRemovalCallback func(manager _IOHIDManagerRef, callback _IOHIDDeviceCallback, context unsafe.Pointer)

var _IOHIDManagerScheduleWithRunLoop func(manager _IOHIDManagerRef, runLoop corefoundation.CFRunLoopRef, runLoopMode corefoundation.CFStringRef)

var _IOHIDElementGetType func(element _IOHIDElementRef) _IOHIDElementType

var _IOHIDElementGetUsage func(element _IOHIDElementRef) uint32

var _IOHIDElementGetUsagePage func(element _IOHIDElementRef) uint32

var _IOHIDElementGetLogicalMin func(element _IOHIDElementRef) corefoundation.CFIndex

var _IOHIDElementGetLogicalMax func(element _IOHIDElementRef) corefoundation.CFIndex

var _IOHIDDeviceGetValue func(device _IOHIDDeviceRef, element _IOHIDElementRef, pValue *_IOHIDValueRef) _IOReturn

var _IOHIDValueGetIntegerValue func(value _IOHIDValueRef) corefoundation.CFIndex

var _IOHIDDeviceCopyMatchingElements func(device _IOHIDDeviceRef, matching corefoundation.CFDictionaryRef, options _IOOptionBits) corefoundation.CFArrayRef

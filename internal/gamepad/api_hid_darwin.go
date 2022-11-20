package gamepad

import (
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

var (
	iokit = purego.Dlopen("IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)

	procIOHIDElementGetTypeID                      = purego.Dlsym(iokit, "IOHIDElementGetTypeID")
	procIOHIDManagerCreate                         = purego.Dlsym(iokit, "IOHIDManagerCreate")
	procIOHIDDeviceGetProperty                     = purego.Dlsym(iokit, "IOHIDDeviceGetProperty")
	procIOHIDManagerOpen                           = purego.Dlsym(iokit, "IOHIDManagerOpen")
	procIOHIDManagerSetDeviceMatchingMultiple      = purego.Dlsym(iokit, "IOHIDManagerSetDeviceMatchingMultiple")
	procIOHIDManagerRegisterDeviceMatchingCallback = purego.Dlsym(iokit, "IOHIDManagerRegisterDeviceMatchingCallback")
	procIOHIDManagerRegisterDeviceRemovalCallback  = purego.Dlsym(iokit, "IOHIDManagerRegisterDeviceRemovalCallback")
	procIOHIDManagerScheduleWithRunLoop            = purego.Dlsym(iokit, "IOHIDManagerScheduleWithRunLoop")
	procIOHIDElementGetType                        = purego.Dlsym(iokit, "IOHIDElementGetType")
	procIOHIDElementGetUsage                       = purego.Dlsym(iokit, "IOHIDElementGetUsage")
	procIOHIDElementGetUsagePage                   = purego.Dlsym(iokit, "IOHIDElementGetUsagePage")
	procIOHIDElementGetLogicalMin                  = purego.Dlsym(iokit, "IOHIDElementGetLogicalMin")
	procIOHIDElementGetLogicalMax                  = purego.Dlsym(iokit, "IOHIDElementGetLogicalMax")
	procIOHIDDeviceGetValue                        = purego.Dlsym(iokit, "IOHIDDeviceGetValue")
	procIOHIDValueGetIntegerValue                  = purego.Dlsym(iokit, "IOHIDValueGetIntegerValue")
	procIOHIDDeviceCopyMatchingElements            = purego.Dlsym(iokit, "IOHIDDeviceCopyMatchingElements")
)

func _IOHIDElementGetTypeID() _CFTypeID {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetTypeID)
	return _CFTypeID(ret)
}

func _IOHIDManagerCreate(allocator _CFAllocatorRef, options _IOOptionBits) _IOHIDManagerRef {
	ret, _, _ := purego.SyscallN(procIOHIDManagerCreate, uintptr(allocator), uintptr(options))
	return _IOHIDManagerRef(ret)
}

func _IOHIDDeviceGetProperty(device _IOHIDDeviceRef, key _CFStringRef) _CFTypeRef {
	ret, _, _ := purego.SyscallN(procIOHIDDeviceGetProperty, uintptr(device), uintptr(key))
	return _CFTypeRef(ret)
}

func _IOHIDManagerOpen(manager _IOHIDManagerRef, options _IOOptionBits) _IOReturn {
	ret, _, _ := purego.SyscallN(procIOHIDManagerOpen, uintptr(manager), uintptr(options))
	return _IOReturn(ret)
}

func _IOHIDManagerSetDeviceMatchingMultiple(manager _IOHIDManagerRef, multiple _CFArrayRef) {
	purego.SyscallN(procIOHIDManagerSetDeviceMatchingMultiple, uintptr(manager), uintptr(multiple))
}

func _IOHIDManagerRegisterDeviceMatchingCallback(manager _IOHIDManagerRef, callback _IOHIDDeviceCallback, context unsafe.Pointer) {
	purego.SyscallN(procIOHIDManagerRegisterDeviceMatchingCallback, uintptr(manager), purego.NewCallback(callback), uintptr(context))
}

func _IOHIDManagerRegisterDeviceRemovalCallback(manager _IOHIDManagerRef, callback _IOHIDDeviceCallback, context unsafe.Pointer) {
	purego.SyscallN(procIOHIDManagerRegisterDeviceRemovalCallback, uintptr(manager), purego.NewCallback(callback), uintptr(context))
}

func _IOHIDManagerScheduleWithRunLoop(manager _IOHIDManagerRef, runLoop _CFRunLoopRef, runLoopMode _CFStringRef) {
	purego.SyscallN(procIOHIDManagerScheduleWithRunLoop, uintptr(manager), uintptr(runLoop), uintptr(runLoopMode))
}

func _IOHIDElementGetType(element _IOHIDElementRef) _IOHIDElementType {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetType, uintptr(element))
	return _IOHIDElementType(ret)
}

func _IOHIDElementGetUsage(element _IOHIDElementRef) uint32 {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetUsage, uintptr(element))
	return uint32(ret)
}

func _IOHIDElementGetUsagePage(element _IOHIDElementRef) uint32 {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetUsagePage, uintptr(element))
	return uint32(ret)
}

func _IOHIDElementGetLogicalMin(element _IOHIDElementRef) _CFIndex {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetLogicalMin, uintptr(element))
	return _CFIndex(ret)
}

func _IOHIDElementGetLogicalMax(element _IOHIDElementRef) _CFIndex {
	ret, _, _ := purego.SyscallN(procIOHIDElementGetLogicalMax, uintptr(element))
	return _CFIndex(ret)
}

func _IOHIDDeviceGetValue(device _IOHIDDeviceRef, element _IOHIDElementRef, pValue *_IOHIDValueRef) _IOReturn {
	if pValue == nil {
		panic("IOHID: pValue cannot be nil")
	}
	ret, _, _ := purego.SyscallN(procIOHIDDeviceGetValue, uintptr(device), uintptr(element), uintptr(unsafe.Pointer(pValue)))
	return _IOReturn(ret)
}

func _IOHIDValueGetIntegerValue(value _IOHIDValueRef) _CFIndex {
	ret, _, _ := purego.SyscallN(procIOHIDValueGetIntegerValue, uintptr(value))
	return _CFIndex(ret)
}

func _IOHIDDeviceCopyMatchingElements(device _IOHIDDeviceRef, matching _CFDictionaryRef, options _IOOptionBits) _CFArrayRef {
	ret, _, _ := purego.SyscallN(procIOHIDDeviceCopyMatchingElements, uintptr(device), uintptr(matching), uintptr(options))
	return _CFArrayRef(ret)
}

// Copyright 2022 The Ebitengine Authors
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

package goglfw

import (
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

type (
	NSOpenGLPixelFormatAttribute uint32
	NSOpenGLContextParameter     cocoa.NSInteger
)

const (
	NSOpenGLPFADoubleBuffer       NSOpenGLPixelFormatAttribute = 5
	NSOpenGLPFAAuxBuffers         NSOpenGLPixelFormatAttribute = 7
	NSOpenGLPFAColorSize          NSOpenGLPixelFormatAttribute = 8
	NSOpenGLPFAAlphaSize          NSOpenGLPixelFormatAttribute = 11
	NSOpenGLPFADepthSize          NSOpenGLPixelFormatAttribute = 12
	NSOpenGLPFAStencilSize        NSOpenGLPixelFormatAttribute = 13
	NSOpenGLPFAAccumSize          NSOpenGLPixelFormatAttribute = 14
	NSOpenGLPFASampleBuffers      NSOpenGLPixelFormatAttribute = 55
	NSOpenGLPFAAccelerated        NSOpenGLPixelFormatAttribute = 73
	NSOpenGLPFAClosestPolicy      NSOpenGLPixelFormatAttribute = 74
	NSOpenGLPFAOpenGLProfile      NSOpenGLPixelFormatAttribute = 99
	NSOpenGLProfileVersion4_1Core NSOpenGLPixelFormatAttribute = 0x4100
	NSOpenGLProfileVersion3_2Core NSOpenGLPixelFormatAttribute = 0x3200
)

const (
	NSOpenGLContextParameterSwapInterval NSOpenGLContextParameter = 222
)

var (
	class_NSOpenGLPixelFormat = objc.GetClass("NSOpenGLPixelFormat")
	class_NSOpenGLContext     = objc.GetClass("NSOpenGLContext")
)

var (
	sel_alloc               = objc.RegisterName("alloc")
	sel_initWithAttributes  = objc.RegisterName("initWithAttributes:")
	sel_clearCurrentContext = objc.RegisterName("clearCurrentContext")
	sel_makeContextCurrent  = objc.RegisterName("makeCurrentContext")
	sel_init                = objc.RegisterName("init")
)

type NSOpenGLPixelFormat struct {
	objc.ID
}

func NSOpenGLPixelFormat_alloc() NSOpenGLPixelFormat {
	return NSOpenGLPixelFormat{objc.ID(class_NSOpenGLPixelFormat).Send(sel_alloc)}
}

func (f NSOpenGLPixelFormat) initWithAttributes(attribs []NSOpenGLPixelFormatAttribute) NSOpenGLPixelFormat {
	if attribs[len(attribs)-1] != 0 {
		panic("glfwwin: attribs must end with null terminator")
	}
	return NSOpenGLPixelFormat{f.Send(sel_initWithAttributes, &attribs[0])}
}

type NSOpenGLContext struct {
	objc.ID
}

func NSOpenGLContext_alloc() NSOpenGLContext {
	return NSOpenGLContext{objc.ID(class_NSOpenGLContext).Send(sel_alloc)}
}

func NSOpenGLContext_clearCurrentContext() {
	objc.ID(class_NSOpenGLContext).Send(sel_clearCurrentContext)
}

func (c NSOpenGLContext) initWithFormat_shareContext(format NSOpenGLPixelFormat, share NSOpenGLContext) NSOpenGLContext {
	return NSOpenGLContext{c.Send(objc.RegisterName("initWithFormat:shareContext:"), format.ID, share.ID)}
}

func (c NSOpenGLContext) makeCurrentContext() {
	c.Send(sel_makeContextCurrent)
}

func (c NSOpenGLContext) setValues_forParameter(vals *int, param NSOpenGLContextParameter) {
	c.Send(objc.RegisterName("setValues:forParameter:"), vals, param)
}

func (c NSOpenGLContext) flushBuffer() {
	c.Send(objc.RegisterName("flushBuffer"))
}

// CString converts a go string to *byte that can be passed to C code.
func CString(name string) []byte {
	if strings.HasSuffix(name, "\x00") {
		return *(*[]byte)(unsafe.Pointer(&name))
	}
	b := make([]byte, len(name)+1)
	copy(b, name)
	return b
}

var (
	_CoreGraphics, _             = purego.Dlopen("CoreGraphics.framework/CoreGraphics", purego.RTLD_GLOBAL)
	procCGGetOnlineDisplayList   uintptr
	procCGDisplayIsAsleep        uintptr
	procCGDisplayUnitNumber      uintptr
	procCGDisplayCopyDisplayMode uintptr
	procCGDisplayModeRelease     uintptr
	procCGDisplayModeGetWidth    uintptr
	procCGDisplayModeGetHeight   uintptr
)

func init() {
	procCGGetOnlineDisplayList, _ = purego.Dlsym(_CoreGraphics, "CGGetOnlineDisplayList")
	procCGDisplayIsAsleep, _ = purego.Dlsym(_CoreGraphics, "CGDisplayIsAsleep")
	procCGDisplayUnitNumber, _ = purego.Dlsym(_CoreGraphics, "CGDisplayUnitNumber")
	procCGDisplayCopyDisplayMode, _ = purego.Dlsym(_CoreGraphics, "CGDisplayCopyDisplayMode")
	procCGDisplayModeRelease, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeRelease")
	procCGDisplayModeGetWidth, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetWidth")
	procCGDisplayModeGetHeight, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetHeight")
	if err := initializeCF(); err != nil {
		panic(err)
	}
}

type (
	_CGDirectDisplayID uint32
	_CGDisplayModeRef  uintptr
	_CGError           uint32
)

func _CGGetOnlineDisplayList(maxDisplays uint32, onlineDisplays *_CGDirectDisplayID, displayCount *uint32) _CGError {
	ret, _, _ := purego.SyscallN(procCGGetOnlineDisplayList, uintptr(maxDisplays), uintptr(unsafe.Pointer(onlineDisplays)), uintptr(unsafe.Pointer(displayCount)))
	return _CGError(ret)
}

func _CGDisplayIsAsleep(display _CGDirectDisplayID) bool {
	ret, _, _ := purego.SyscallN(procCGDisplayIsAsleep, uintptr(display))
	return ret != 0
}

func _CGDisplayUnitNumber(display _CGDirectDisplayID) uint32 {
	ret, _, _ := purego.SyscallN(procCGDisplayUnitNumber, uintptr(display))
	return uint32(ret)
}

var _CGDisplayBounds func(display _CGDirectDisplayID) cocoa.CGRect

func _CGDisplayCopyDisplayMode(display _CGDirectDisplayID) _CGDisplayModeRef {
	ret, _, _ := purego.SyscallN(procCGDisplayCopyDisplayMode, uintptr(display))
	return _CGDisplayModeRef(ret)
}

func _CGDisplayModeRelease(mode _CGDisplayModeRef) {
	purego.SyscallN(procCGDisplayModeRelease, uintptr(mode))
}

func _CGDisplayModeGetWidth(mode _CGDisplayModeRef) uintptr {
	ret, _, _ := purego.SyscallN(procCGDisplayModeGetWidth, uintptr(mode))
	return ret
}

func _CGDisplayModeGetHeight(mode _CGDisplayModeRef) uintptr {
	ret, _, _ := purego.SyscallN(procCGDisplayModeGetHeight, uintptr(mode))
	return ret
}

var _CGDisplayModeGetRefreshRate func(mode _CGDisplayModeRef) float64

func init() {
	purego.RegisterLibFunc(&_CGDisplayModeGetRefreshRate, _CoreGraphics, "CGDisplayModeGetRefreshRate")
	purego.RegisterLibFunc(&_CGDisplayBounds, _CoreGraphics, "CGDisplayBounds")
}

// This code is mostly copied from api_cf_darwin.go with some additional functions
type (
	_CFIndex                    int64
	_CFAllocatorRef             uintptr
	_CFArrayRef                 uintptr
	_CFDictionaryRef            uintptr
	_CFNumberRef                uintptr
	_CFTypeRef                  uintptr
	_CFRunLoopRef               uintptr
	_CFNumberType               uintptr
	_CFStringRef                uintptr
	_CFBundleRef                uintptr
	_CFArrayCallBacks           struct{}
	_CFDictionaryKeyCallBacks   struct{}
	_CFDictionaryValueCallBacks struct{}
	_CFRunLoopRunResult         int32
	_CFRunLoopMode              = _CFStringRef
	_CFTimeInterval             float64
	_CFTypeID                   uint64
	_CFStringEncoding           uint32
)

var kCFAllocatorDefault _CFAllocatorRef = 0

const (
	kCFStringEncodingUTF8 _CFStringEncoding = 0x08000100
)

const (
	kCFNumberSInt32Type _CFNumberType = 3
	kCFNumberIntType    _CFNumberType = 9
)

var (
	kCFTypeDictionaryKeyCallBacks   uintptr
	kCFTypeDictionaryValueCallBacks uintptr
	kCFTypeArrayCallBacks           uintptr
	kCFRunLoopDefaultMode           uintptr
)

func initializeCF() error {
	corefoundation, err := purego.Dlopen("CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}

	kCFTypeDictionaryKeyCallBacks, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryKeyCallBacks")
	if err != nil {
		return err
	}
	kCFTypeDictionaryValueCallBacks, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryValueCallBacks")
	if err != nil {
		return err
	}
	kCFTypeArrayCallBacks, err = purego.Dlsym(corefoundation, "kCFTypeArrayCallBacks")
	if err != nil {
		return err
	}
	kCFRunLoopDefaultMode, err = purego.Dlsym(corefoundation, "kCFRunLoopDefaultMode")
	if err != nil {
		return err
	}

	purego.RegisterLibFunc(&_CFNumberCreate, corefoundation, "CFNumberCreate")
	purego.RegisterLibFunc(&_CFNumberGetValue, corefoundation, "CFNumberGetValue")
	purego.RegisterLibFunc(&_CFArrayCreate, corefoundation, "CFArrayCreate")
	purego.RegisterLibFunc(&_CFArrayGetValueAtIndex, corefoundation, "CFArrayGetValueAtIndex")
	purego.RegisterLibFunc(&_CFArrayGetCount, corefoundation, "CFArrayGetCount")
	purego.RegisterLibFunc(&_CFDictionaryCreate, corefoundation, "CFDictionaryCreate")
	purego.RegisterLibFunc(&_CFRelease, corefoundation, "CFRelease")
	purego.RegisterLibFunc(&_CFRunLoopGetMain, corefoundation, "CFRunLoopGetMain")
	purego.RegisterLibFunc(&_CFRunLoopRunInMode, corefoundation, "CFRunLoopRunInMode")
	purego.RegisterLibFunc(&_CFGetTypeID, corefoundation, "CFGetTypeID")
	purego.RegisterLibFunc(&_CFStringGetCString, corefoundation, "CFStringGetCString")
	purego.RegisterLibFunc(&_CFStringCreateWithCString, corefoundation, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&_CFBundleGetBundleWithIdentifier, corefoundation, "CFBundleGetBundleWithIdentifier")
	purego.RegisterLibFunc(&_CFBundleGetFunctionPointerForName, corefoundation, "CFBundleGetFunctionPointerForName")

	return nil
}

var (
	_CFNumberCreate                    func(allocator _CFAllocatorRef, theType _CFNumberType, valuePtr unsafe.Pointer) _CFNumberRef
	_CFNumberGetValue                  func(number _CFNumberRef, theType _CFNumberType, valuePtr unsafe.Pointer) bool
	_CFArrayCreate                     func(allocator _CFAllocatorRef, values *unsafe.Pointer, numValues _CFIndex, callbacks *_CFArrayCallBacks) _CFArrayRef
	_CFArrayGetValueAtIndex            func(array _CFArrayRef, index _CFIndex) uintptr
	_CFArrayGetCount                   func(array _CFArrayRef) _CFIndex
	_CFDictionaryCreate                func(allocator _CFAllocatorRef, keys *unsafe.Pointer, values *unsafe.Pointer, numValues _CFIndex, keyCallBacks *_CFDictionaryKeyCallBacks, valueCallBacks *_CFDictionaryValueCallBacks) _CFDictionaryRef
	_CFRelease                         func(cf _CFTypeRef)
	_CFRunLoopGetMain                  func() _CFRunLoopRef
	_CFRunLoopRunInMode                func(mode _CFRunLoopMode, seconds _CFTimeInterval, returnAfterSourceHandled bool) _CFRunLoopRunResult
	_CFGetTypeID                       func(cf _CFTypeRef) _CFTypeID
	_CFStringGetCString                func(theString _CFStringRef, buffer []byte, encoding _CFStringEncoding) bool
	_CFStringCreateWithCString         func(alloc _CFAllocatorRef, cstr []byte, encoding _CFStringEncoding) _CFStringRef
	_CFBundleGetBundleWithIdentifier   func(bundleID _CFStringRef) _CFBundleRef
	_CFBundleGetFunctionPointerForName func(bundle _CFBundleRef, functionName _CFStringRef) uintptr
)

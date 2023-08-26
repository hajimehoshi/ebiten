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

package gamepad

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

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
	corefoundation, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
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

	return nil
}

var (
	_CFNumberCreate            func(allocator _CFAllocatorRef, theType _CFNumberType, valuePtr unsafe.Pointer) _CFNumberRef
	_CFNumberGetValue          func(number _CFNumberRef, theType _CFNumberType, valuePtr unsafe.Pointer) bool
	_CFArrayCreate             func(allocator _CFAllocatorRef, values *unsafe.Pointer, numValues _CFIndex, callbacks *_CFArrayCallBacks) _CFArrayRef
	_CFArrayGetValueAtIndex    func(array _CFArrayRef, index _CFIndex) uintptr
	_CFArrayGetCount           func(array _CFArrayRef) _CFIndex
	_CFDictionaryCreate        func(allocator _CFAllocatorRef, keys *unsafe.Pointer, values *unsafe.Pointer, numValues _CFIndex, keyCallBacks *_CFDictionaryKeyCallBacks, valueCallBacks *_CFDictionaryValueCallBacks) _CFDictionaryRef
	_CFRelease                 func(cf _CFTypeRef)
	_CFRunLoopGetMain          func() _CFRunLoopRef
	_CFRunLoopRunInMode        func(mode _CFRunLoopMode, seconds _CFTimeInterval, returnAfterSourceHandled bool) _CFRunLoopRunResult
	_CFGetTypeID               func(cf _CFTypeRef) _CFTypeID
	_CFStringGetCString        func(theString _CFStringRef, buffer []byte, encoding _CFStringEncoding) bool
	_CFStringCreateWithCString func(alloc _CFAllocatorRef, cstr []byte, encoding _CFStringEncoding) _CFStringRef
)

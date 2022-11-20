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

type _CFIndex int64
type _CFAllocatorRef uintptr
type _CFArrayRef uintptr
type _CFDictionaryRef uintptr
type _CFNumberRef uintptr
type _CFTypeRef uintptr
type _CFRunLoopRef uintptr
type _CFNumberType uintptr
type _CFStringRef uintptr
type _CFArrayCallBacks struct{}
type _CFDictionaryKeyCallBacks struct{}
type _CFDictionaryValueCallBacks struct{}
type _CFRunLoopRunResult int32
type _CFRunLoopMode = _CFStringRef
type _CFTimeInterval float64
type _CFTypeID uint64
type _CFStringEncoding uint32

var kCFAllocatorDefault _CFAllocatorRef = 0

const (
	kCFStringEncodingUTF8 _CFStringEncoding = 0x08000100
)

const (
	kCFNumberSInt32Type _CFNumberType = 3
	kCFNumberIntType    _CFNumberType = 9
)

var (
	corefoundation = purego.Dlopen("CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)

	kCFTypeDictionaryKeyCallBacks   = purego.Dlsym(corefoundation, "kCFTypeDictionaryKeyCallBacks")
	kCFTypeDictionaryValueCallBacks = purego.Dlsym(corefoundation, "kCFTypeDictionaryValueCallBacks")
	kCFTypeArrayCallBacks           = purego.Dlsym(corefoundation, "kCFTypeArrayCallBacks")
	kCFRunLoopDefaultMode           = purego.Dlsym(corefoundation, "kCFRunLoopDefaultMode")
)

func init() {
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
}

var _CFNumberCreate func(allocator _CFAllocatorRef, theType _CFNumberType, valuePtr unsafe.Pointer) _CFNumberRef

var _CFNumberGetValue func(number _CFNumberRef, theType _CFNumberType, valuePtr unsafe.Pointer) bool

var _CFArrayCreate func(allocator _CFAllocatorRef, values *unsafe.Pointer, numValues _CFIndex, callbacks *_CFArrayCallBacks) _CFArrayRef

var _CFArrayGetValueAtIndex func(array _CFArrayRef, index _CFIndex) uintptr

var _CFArrayGetCount func(array _CFArrayRef) _CFIndex

var _CFDictionaryCreate func(allocator _CFAllocatorRef, keys *unsafe.Pointer, values *unsafe.Pointer, numValues _CFIndex, keyCallBacks *_CFDictionaryKeyCallBacks, valueCallBacks *_CFDictionaryValueCallBacks) _CFDictionaryRef

var _CFRelease func(cf _CFTypeRef)

var _CFRunLoopGetMain func() _CFRunLoopRef

var _CFRunLoopRunInMode func(mode _CFRunLoopMode, seconds _CFTimeInterval, returnAfterSourceHandled bool) _CFRunLoopRunResult

var _CFGetTypeID func(cf _CFTypeRef) _CFTypeID

var _CFStringGetCString func(theString _CFStringRef, buffer []byte, encoding _CFStringEncoding) bool

var _CFStringCreateWithCString func(alloc _CFAllocatorRef, cstr []byte, encoding _CFStringEncoding) _CFStringRef

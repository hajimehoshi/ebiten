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

package corefoundation

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

type CFIndex int64
type CFAllocatorRef uintptr
type CFArrayRef uintptr
type CFDictionaryRef uintptr
type CFNumberRef uintptr
type CFTypeRef uintptr
type CFRunLoopRef uintptr
type CFNumberType uintptr
type CFStringRef uintptr
type CFArrayCallBacks struct{}
type CFDictionaryKeyCallBacks struct{}
type CFDictionaryValueCallBacks struct{}
type CFRunLoopRunResult int32
type CFRunLoopMode = CFStringRef
type CFTimeInterval float64
type CFTypeID uint64
type CFStringEncoding uint32
type CFBundleRef uintptr

var KCFAllocatorDefault CFAllocatorRef = 0

const (
	KCFStringEncodingUTF8 CFStringEncoding = 0x08000100
)

const (
	KCFNumberSInt32Type CFNumberType = 3
	KCFNumberIntType    CFNumberType = 9
)

var (
	KCFTypeDictionaryKeyCallBacks   *CFDictionaryKeyCallBacks
	KCFTypeDictionaryValueCallBacks *CFDictionaryValueCallBacks
	KCFTypeArrayCallBacks           *CFArrayCallBacks
	KCFRunLoopDefaultMode           CFRunLoopMode
)

func InitializeCF() error {
	corefoundation, err := purego.Dlopen("CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	var ptr uintptr
	ptr, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryKeyCallBacks")
	if err != nil {
		return err
	}
	KCFTypeDictionaryKeyCallBacks = (*CFDictionaryKeyCallBacks)(unsafe.Pointer(ptr))

	ptr, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryValueCallBacks")
	if err != nil {
		return err
	}
	KCFTypeDictionaryValueCallBacks = (*CFDictionaryValueCallBacks)(unsafe.Pointer(ptr))

	ptr, err = purego.Dlsym(corefoundation, "kCFTypeArrayCallBacks")
	if err != nil {
		return err
	}
	KCFTypeArrayCallBacks = (*CFArrayCallBacks)(unsafe.Pointer(ptr))

	ptr, err = purego.Dlsym(corefoundation, "kCFRunLoopDefaultMode")
	if err != nil {
		return err
	}
	KCFRunLoopDefaultMode = (CFRunLoopMode)(unsafe.Pointer(ptr))

	purego.RegisterLibFunc(&CFNumberCreate, corefoundation, "CFNumberCreate")
	purego.RegisterLibFunc(&CFNumberGetValue, corefoundation, "CFNumberGetValue")
	purego.RegisterLibFunc(&CFArrayCreate, corefoundation, "CFArrayCreate")
	purego.RegisterLibFunc(&CFArrayGetValueAtIndex, corefoundation, "CFArrayGetValueAtIndex")
	purego.RegisterLibFunc(&CFArrayGetCount, corefoundation, "CFArrayGetCount")
	purego.RegisterLibFunc(&CFDictionaryCreate, corefoundation, "CFDictionaryCreate")
	purego.RegisterLibFunc(&CFRelease, corefoundation, "CFRelease")
	purego.RegisterLibFunc(&CFRunLoopGetMain, corefoundation, "CFRunLoopGetMain")
	purego.RegisterLibFunc(&CFRunLoopRunInMode, corefoundation, "CFRunLoopRunInMode")
	purego.RegisterLibFunc(&CFGetTypeID, corefoundation, "CFGetTypeID")
	purego.RegisterLibFunc(&CFStringGetCString, corefoundation, "CFStringGetCString")
	purego.RegisterLibFunc(&CFStringCreateWithCString, corefoundation, "CFStringCreateWithCString")

	return nil
}

var CFNumberCreate func(allocator CFAllocatorRef, theType CFNumberType, valuePtr unsafe.Pointer) CFNumberRef

var CFNumberGetValue func(number CFNumberRef, theType CFNumberType, valuePtr unsafe.Pointer) bool

var CFArrayCreate func(allocator CFAllocatorRef, values *unsafe.Pointer, numValues CFIndex, callbacks *CFArrayCallBacks) CFArrayRef

var CFArrayGetValueAtIndex func(array CFArrayRef, index CFIndex) uintptr

var CFArrayGetCount func(array CFArrayRef) CFIndex

var CFDictionaryCreate func(allocator CFAllocatorRef, keys *unsafe.Pointer, values *unsafe.Pointer, numValues CFIndex, keyCallBacks *CFDictionaryKeyCallBacks, valueCallBacks *CFDictionaryValueCallBacks) CFDictionaryRef

var CFRelease func(cf CFTypeRef)

var CFRunLoopGetMain func() CFRunLoopRef

var CFRunLoopRunInMode func(mode CFRunLoopMode, seconds CFTimeInterval, returnAfterSourceHandled bool) CFRunLoopRunResult

var CFGetTypeID func(cf CFTypeRef) CFTypeID

var CFStringGetCString func(theString CFStringRef, buffer []byte, encoding CFStringEncoding) bool

var CFStringCreateWithCString func(alloc CFAllocatorRef, cstr []byte, encoding CFStringEncoding) CFStringRef

var CFBundleGetBundleWithIdentifier func(bundleID CFStringRef) CFBundleRef

var CFBundleGetFunctionPointerForName func(bundle CFBundleRef, functionName CFStringRef) uintptr

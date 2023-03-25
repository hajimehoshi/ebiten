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
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

type (
	CFIndex                    int64
	CFAllocatorRef             uintptr
	CFArrayRef                 uintptr
	CFDictionaryRef            uintptr
	CFNumberRef                uintptr
	CFTypeRef                  uintptr
	CFRunLoopRef               uintptr
	CFNumberType               uintptr
	CFStringRef                uintptr
	CFArrayCallBacks           struct{}
	CFDictionaryKeyCallBacks   struct{}
	CFDictionaryValueCallBacks struct{}
	CFRunLoopRunResult         int32
	CFRunLoopMode              = CFStringRef
	CFTimeInterval             float64
	CFTypeID                   uint64
	CFStringEncoding           uint32
	CFBundleRef                uintptr
)

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

var once sync.Once

func InitializeCF() (err error) {
	once.Do(func() {
		var corefoundation uintptr
		corefoundation, err = purego.Dlopen("CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			return
		}
		var ptr uintptr
		ptr, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryKeyCallBacks")
		if err != nil {
			return
		}
		KCFTypeDictionaryKeyCallBacks = (*CFDictionaryKeyCallBacks)(unsafe.Pointer(ptr))

		ptr, err = purego.Dlsym(corefoundation, "kCFTypeDictionaryValueCallBacks")
		if err != nil {
			return
		}
		KCFTypeDictionaryValueCallBacks = (*CFDictionaryValueCallBacks)(unsafe.Pointer(ptr))

		ptr, err = purego.Dlsym(corefoundation, "kCFTypeArrayCallBacks")
		if err != nil {
			return
		}
		KCFTypeArrayCallBacks = (*CFArrayCallBacks)(unsafe.Pointer(ptr))

		ptr, err = purego.Dlsym(corefoundation, "kCFRunLoopDefaultMode")
		if err != nil {
			return
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
		purego.RegisterLibFunc(&CFBundleGetBundleWithIdentifier, corefoundation, "CFBundleGetBundleWithIdentifier")
		purego.RegisterLibFunc(&CFBundleGetFunctionPointerForName, corefoundation, "CFBundleGetFunctionPointerForName")
	})
	return err
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

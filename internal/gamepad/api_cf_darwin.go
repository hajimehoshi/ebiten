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
	corefoundation                = purego.Dlopen("CoreFoundation.framework/CoreFoundation", purego.RTLD_GLOBAL)
	procCFNumberCreate            = purego.Dlsym(corefoundation, "CFNumberCreate")
	procCFNumberGetValue          = purego.Dlsym(corefoundation, "CFNumberGetValue")
	procCFArrayCreate             = purego.Dlsym(corefoundation, "CFArrayCreate")
	procCFArrayGetCount           = purego.Dlsym(corefoundation, "CFArrayGetCount")
	procCFArrayGetValueAtIndex    = purego.Dlsym(corefoundation, "CFArrayGetValueAtIndex")
	procCFDictionaryCreate        = purego.Dlsym(corefoundation, "CFDictionaryCreate")
	procCFRelease                 = purego.Dlsym(corefoundation, "CFRelease")
	procCFRunLoopGetMain          = purego.Dlsym(corefoundation, "CFRunLoopGetMain")
	procCFRunLoopRunInMode        = purego.Dlsym(corefoundation, "CFRunLoopRunInMode")
	procCFGetTypeID               = purego.Dlsym(corefoundation, "CFGetTypeID")
	procCFStringGetCString        = purego.Dlsym(corefoundation, "CFStringGetCString")
	procCFStringCreateWithCString = purego.Dlsym(corefoundation, "CFStringCreateWithCString")

	kCFTypeDictionaryKeyCallBacks   = purego.Dlsym(corefoundation, "kCFTypeDictionaryKeyCallBacks")
	kCFTypeDictionaryValueCallBacks = purego.Dlsym(corefoundation, "kCFTypeDictionaryValueCallBacks")
	kCFTypeArrayCallBacks           = purego.Dlsym(corefoundation, "kCFTypeArrayCallBacks")
	kCFRunLoopDefaultMode           = purego.Dlsym(corefoundation, "kCFRunLoopDefaultMode")
)

func _CFNumberCreate(allocator _CFAllocatorRef, theType _CFNumberType, valuePtr unsafe.Pointer) _CFNumberRef {
	number, _, _ := purego.SyscallN(procCFNumberCreate, uintptr(allocator), uintptr(theType), uintptr(valuePtr))
	return _CFNumberRef(number)
}

func _CFNumberGetValue(number _CFNumberRef, theType _CFNumberType, valuePtr unsafe.Pointer) bool {
	ret, _, _ := purego.SyscallN(procCFNumberGetValue, uintptr(number), uintptr(theType), uintptr(valuePtr))
	return ret != 0
}

func _CFArrayCreate(allocator _CFAllocatorRef, values *unsafe.Pointer, numValues _CFIndex, callbacks *_CFArrayCallBacks) _CFArrayRef {
	ret, _, _ := purego.SyscallN(procCFArrayCreate, uintptr(allocator), uintptr(unsafe.Pointer(values)), uintptr(numValues), uintptr(unsafe.Pointer(callbacks)))
	return _CFArrayRef(ret)
}

func _CFArrayGetValueAtIndex(array _CFArrayRef, index _CFIndex) uintptr {
	ret, _, _ := purego.SyscallN(procCFArrayGetValueAtIndex, uintptr(array), uintptr(index))
	return ret
}

func _CFArrayGetCount(array _CFArrayRef) _CFIndex {
	ret, _, _ := purego.SyscallN(procCFArrayGetCount, uintptr(array))
	return _CFIndex(ret)
}

func _CFDictionaryCreate(allocator _CFAllocatorRef, keys *unsafe.Pointer, values *unsafe.Pointer, numValues _CFIndex, keyCallBacks *_CFDictionaryKeyCallBacks, valueCallBacks *_CFDictionaryValueCallBacks) _CFDictionaryRef {
	ret, _, _ := purego.SyscallN(procCFDictionaryCreate, uintptr(allocator), uintptr(unsafe.Pointer(keys)), uintptr(unsafe.Pointer(values)), uintptr(numValues), uintptr(unsafe.Pointer(keyCallBacks)), uintptr(unsafe.Pointer(valueCallBacks)))
	return _CFDictionaryRef(ret)
}

func _CFRelease(cf _CFTypeRef) {
	purego.SyscallN(procCFRelease, uintptr(cf))
}

func _CFRunLoopGetMain() _CFRunLoopRef {
	ret, _, _ := purego.SyscallN(procCFRunLoopGetMain)
	return _CFRunLoopRef(ret)
}

func _CFRunLoopRunInMode(mode _CFRunLoopMode, seconds _CFTimeInterval, returnAfterSourceHandled bool) _CFRunLoopRunResult {
	var b uintptr = 0
	if returnAfterSourceHandled {
		b = 1
	}
	if seconds != 0 {
		panic("corefoundation: seconds greater than 0 is not supported")
	}
	//TODO: support floats
	ret, _, _ := purego.SyscallN(procCFRunLoopRunInMode, uintptr(mode), b)
	return _CFRunLoopRunResult(ret)
}

func _CFGetTypeID(cf _CFTypeRef) _CFTypeID {
	ret, _, _ := purego.SyscallN(procCFGetTypeID, uintptr(cf))
	return _CFTypeID(ret)
}

func _CFStringGetCString(theString _CFStringRef, buffer []byte, encoding _CFStringEncoding) bool {
	ret, _, _ := purego.SyscallN(procCFStringGetCString, uintptr(theString), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(encoding))
	return ret != 0
}

func _CFStringCreateWithCString(alloc _CFAllocatorRef, cstr []byte, encoding _CFStringEncoding) _CFStringRef {
	ret, _, _ := purego.SyscallN(procCFStringCreateWithCString, uintptr(alloc), uintptr(unsafe.Pointer(&cstr[0])), uintptr(encoding))
	return _CFStringRef(ret)
}

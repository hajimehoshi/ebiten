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

package cocoa

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	class_NSInvocation      = objc.GetClass("NSInvocation")
	class_NSMethodSignature = objc.GetClass("NSMethodSignature")
	class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	class_NSString          = objc.GetClass("NSString")
)

var (
	sel_alloc                              = objc.RegisterName("alloc")
	sel_new                                = objc.RegisterName("new")
	sel_release                            = objc.RegisterName("release")
	sel_invocationWithMethodSignature      = objc.RegisterName("invocationWithMethodSignature:")
	sel_setSelector                        = objc.RegisterName("setSelector:")
	sel_setTarget                          = objc.RegisterName("setTarget:")
	sel_setArgumentAtIndex                 = objc.RegisterName("setArgument:atIndex:")
	sel_getReturnValue                     = objc.RegisterName("getReturnValue:")
	sel_invoke                             = objc.RegisterName("invoke")
	sel_instanceMethodSignatureForSelector = objc.RegisterName("instanceMethodSignatureForSelector:")
	sel_signatureWithObjCTypes             = objc.RegisterName("signatureWithObjCTypes:")
	sel_initWithUTF8String                 = objc.RegisterName("initWithUTF8String:")
	sel_UTF8String                         = objc.RegisterName("UTF8String")
	sel_length                             = objc.RegisterName("length")
)

type CGFloat float64

type CGSize struct {
	Width, Height CGFloat
}

type NSError struct {
	objc.ID
}

// NSInvocation is being used to call functions that can't be called directly with purego.SyscallN.
// See the downsides of that function for what it cannot do.
type NSInvocation struct {
	objc.ID
}

func NSInvocation_invocationWithMethodSignature(sig NSMethodSignature) NSInvocation {
	return NSInvocation{objc.ID(class_NSInvocation).Send(sel_invocationWithMethodSignature, sig.ID)}
}

func (inv NSInvocation) SetSelector(_cmd objc.SEL) {
	inv.Send(sel_setSelector, _cmd)
}

func (inv NSInvocation) SetTarget(target objc.ID) {
	inv.Send(sel_setTarget, target)
}

func (inv NSInvocation) SetArgumentAtIndex(arg unsafe.Pointer, idx int) {
	inv.Send(sel_setArgumentAtIndex, arg, idx)
}

func (inv NSInvocation) GetReturnValue(ret unsafe.Pointer) {
	inv.Send(sel_getReturnValue, ret)
}

func (inv NSInvocation) Invoke() {
	inv.Send(sel_invoke)
}

type NSMethodSignature struct {
	objc.ID
}

func NSMethodSignature_instanceMethodSignatureForSelector(self objc.ID, _cmd objc.SEL) NSMethodSignature {
	return NSMethodSignature{self.Send(sel_instanceMethodSignatureForSelector, _cmd)}
}

// NSMethodSignature_signatureWithObjCTypes takes a string that represents the type signature of a method.
// It follows the encoding specified in the Apple Docs.
//
// [Apple Docs]: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/ObjCRuntimeGuide/Articles/ocrtTypeEncodings.html#//apple_ref/doc/uid/TP40008048-CH100
func NSMethodSignature_signatureWithObjCTypes(types string) NSMethodSignature {
	return NSMethodSignature{objc.ID(class_NSMethodSignature).Send(sel_signatureWithObjCTypes, types)}
}

type NSAutoreleasePool struct {
	objc.ID
}

func NSAutoreleasePool_new() NSAutoreleasePool {
	return NSAutoreleasePool{objc.ID(class_NSAutoreleasePool).Send(sel_new)}
}

func (pool NSAutoreleasePool) Release() {
	pool.Send(sel_release)
}

type NSString struct {
	objc.ID
}

func NSString_alloc() NSString {
	return NSString{objc.ID(class_NSString).Send(sel_alloc)}
}

func (s NSString) InitWithUTF8String(utf8 string) NSString {
	return NSString{s.Send(sel_initWithUTF8String, utf8)}
}

func (s NSString) String() string {
	// this will be nicer with unsafe.Slice once ebitengine requires 1.17
	// reflect.SliceHeader is used because it will force Go to copy the string
	// into Go memory when casted to a string
	var b []byte
	header := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	header.Data = uintptr(s.Send(sel_UTF8String))
	header.Len = int(s.Send(sel_length))
	header.Cap = header.Len
	return string(b)
}

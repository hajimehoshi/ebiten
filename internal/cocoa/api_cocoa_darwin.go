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
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	class_NSInvocation      = objc.GetClass("NSInvocation")
	class_NSMethodSignature = objc.GetClass("NSMethodSignature")
	class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	class_NSString          = objc.GetClass("NSString")
	class_NSProcessInfo     = objc.GetClass("NSProcessInfo")
	class_NSColor           = objc.GetClass("NSColor")
	class_NSWindow          = objc.GetClass("NSWindow")
	class_NSView            = objc.GetClass("NSView")
	class_NSScreen          = objc.GetClass("NSScreen")
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
	sel_invokeWithTarget                   = objc.RegisterName("invokeWithTarget:")
	sel_instanceMethodSignatureForSelector = objc.RegisterName("instanceMethodSignatureForSelector:")
	sel_signatureWithObjCTypes             = objc.RegisterName("signatureWithObjCTypes:")
	sel_initWithUTF8String                 = objc.RegisterName("initWithUTF8String:")
	sel_UTF8String                         = objc.RegisterName("UTF8String")
	sel_length                             = objc.RegisterName("length")
	sel_processInfo                        = objc.RegisterName("processInfo")
	sel_frame                              = objc.RegisterName("frame")
	sel_contentView                        = objc.RegisterName("contentView")
	sel_setBackgroundColor                 = objc.RegisterName("setBackgroundColor:")
	sel_colorWithSRGBRedGreenBlueAlpha     = objc.RegisterName("colorWithSRGBRed:green:blue:alpha:")
	sel_setFrameSize                       = objc.RegisterName("setFrameSize:")
	sel_object                             = objc.RegisterName("object")
	sel_styleMask                          = objc.RegisterName("styleMask")
	sel_setStyleMask                       = objc.RegisterName("setStyleMask:")
	sel_mainScreen                         = objc.RegisterName("mainScreen")
	sel_screen                             = objc.RegisterName("screen")
	sel_isVisible                          = objc.RegisterName("isVisible")
	sel_deviceDescription                  = objc.RegisterName("deviceDescription")
	sel_objectForKey                       = objc.RegisterName("objectForKey:")
	sel_unsignedIntValue                   = objc.RegisterName("unsignedIntValue")
)

const (
	NSWindowCollectionBehaviorManaged           = 1 << 2
	NSWindowCollectionBehaviorFullScreenPrimary = 1 << 7
	NSWindowCollectionBehaviorFullScreenNone    = 1 << 9
)

const (
	NSWindowStyleMaskResizable  = 1 << 3
	NSWindowStyleMaskFullScreen = 1 << 14
)

type CGFloat = float64

type CGSize struct {
	Width, Height CGFloat
}

type CGPoint struct {
	X, Y float64
}

type CGRect struct {
	Origin CGPoint
	Size   CGSize
}

type NSUInteger = uint
type NSInteger = int

type NSPoint = CGPoint
type NSRect = CGRect
type NSSize = CGSize

type NSError struct {
	objc.ID
}

type NSColor struct {
	objc.ID
}

func NSColor_colorWithSRGBRedGreenBlueAlpha(red, green, blue, alpha CGFloat) (color NSColor) {
	return NSColor{objc.ID(class_NSColor).Send(sel_colorWithSRGBRedGreenBlueAlpha, red, green, blue, alpha)}
}

type NSOperatingSystemVersion struct {
	Major, Minor, Patch NSInteger
}

type NSProcessInfo struct {
	objc.ID
}

func NSProcessInfo_processInfo() NSProcessInfo {
	return NSProcessInfo{objc.ID(class_NSProcessInfo).Send(sel_processInfo)}
}

type NSWindow struct {
	objc.ID
}

func (w NSWindow) StyleMask() NSUInteger {
	return NSUInteger(w.Send(sel_styleMask))
}

func (w NSWindow) SetStyleMask(styleMask NSUInteger) {
	w.Send(sel_setStyleMask, styleMask)
}

func (w NSWindow) SetBackgroundColor(color NSColor) {
	w.Send(sel_setBackgroundColor, color.ID)
}

func (w NSWindow) IsVisible() bool {
	return w.Send(sel_isVisible) != 0
}

func (w NSWindow) Screen() NSScreen {
	return NSScreen{w.Send(sel_screen)}
}

func (w NSWindow) Frame() NSRect {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSWindow), sel_frame)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(w.ID)
	inv.SetSelector(sel_frame)
	inv.Invoke()
	var rect NSRect
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
}

func (w NSWindow) ContentView() NSView {
	return NSView{w.Send(sel_contentView)}
}

type NSView struct {
	objc.ID
}

func (v NSView) SetFrameSize(size CGSize) {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSView), sel_setFrameSize)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_setFrameSize)
	inv.SetArgumentAtIndex(unsafe.Pointer(&size), 2)
	inv.InvokeWithTarget(v.ID)
}

func (v NSView) Frame() NSRect {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSView), sel_frame)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_frame)
	inv.InvokeWithTarget(v.ID)
	var rect NSRect
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
}

func (v NSView) SetLayer(layer uintptr) {
	v.Send(objc.RegisterName("setLayer:"), layer)
}

func (v NSView) SetWantsLayer(wantsLayer bool) {
	v.Send(objc.RegisterName("setWantsLayer:"), wantsLayer)
}

// NSInvocation is being used to call functions that can't be called directly with purego.SyscallN.
// See the downsides of that function for what it cannot do.
type NSInvocation struct {
	objc.ID
}

func NSInvocation_invocationWithMethodSignature(sig NSMethodSignature) NSInvocation {
	return NSInvocation{objc.ID(class_NSInvocation).Send(sel_invocationWithMethodSignature, sig.ID)}
}

func (i NSInvocation) SetSelector(cmd objc.SEL) {
	i.Send(sel_setSelector, cmd)
}

func (i NSInvocation) SetTarget(target objc.ID) {
	i.Send(sel_setTarget, target)
}

func (i NSInvocation) SetArgumentAtIndex(arg unsafe.Pointer, idx int) {
	i.Send(sel_setArgumentAtIndex, arg, idx)
}

func (i NSInvocation) GetReturnValue(ret unsafe.Pointer) {
	i.Send(sel_getReturnValue, ret)
}

func (i NSInvocation) Invoke() {
	i.Send(sel_invoke)
}

func (i NSInvocation) InvokeWithTarget(target objc.ID) {
	i.Send(sel_invokeWithTarget, target)
}

type NSMethodSignature struct {
	objc.ID
}

func NSMethodSignature_instanceMethodSignatureForSelector(self objc.ID, cmd objc.SEL) NSMethodSignature {
	return NSMethodSignature{self.Send(sel_instanceMethodSignatureForSelector, cmd)}
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

func (p NSAutoreleasePool) Release() {
	p.Send(sel_release)
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
	return string(unsafe.Slice((*byte)(unsafe.Pointer(s.Send(sel_UTF8String))), s.Send(sel_length)))
}

type NSNotification struct {
	objc.ID
}

func (n NSNotification) Object() objc.ID {
	return n.Send(sel_object)
}

type NSScreen struct {
	objc.ID
}

func NSScreen_mainScreen() NSScreen {
	return NSScreen{objc.ID(class_NSScreen).Send(sel_mainScreen)}
}

func (s NSScreen) DeviceDescription() NSDictionary {
	return NSDictionary{s.Send(sel_deviceDescription)}
}

type NSDictionary struct {
	objc.ID
}

func (d NSDictionary) ObjectForKey(object objc.ID) objc.ID {
	return d.Send(sel_objectForKey, object)
}

type NSNumber struct {
	objc.ID
}

func (n NSNumber) UnsignedIntValue() uint {
	return uint(n.Send(sel_unsignedIntValue))
}

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
	"math"
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var Cocoa = purego.Dlopen("Cocoa.framework/Cocoa", purego.RTLD_GLOBAL)

var (
	class_NSInvocation      = objc.GetClass("NSInvocation")
	class_NSMethodSignature = objc.GetClass("NSMethodSignature")
	class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	class_NSString          = objc.GetClass("NSString")
	class_NSProcessInfo     = objc.GetClass("NSProcessInfo")
	class_NSColor           = objc.GetClass("NSColor")
	class_NSCursor          = objc.GetClass("NSCursor")
	class_NSWindow          = objc.GetClass("NSWindow")
	class_NSView            = objc.GetClass("NSView")
	class_NSScreen          = objc.GetClass("NSScreen")
	class_NSThread          = objc.GetClass("NSThread")
	class_NSApplication     = objc.GetClass("NSApplication")
	class_NSDate            = objc.GetClass("NSDate")
)

var (
	sel_alloc                                          = objc.RegisterName("alloc")
	sel_new                                            = objc.RegisterName("new")
	sel_release                                        = objc.RegisterName("release")
	sel_retain                                         = objc.RegisterName("retain")
	sel_invocationWithMethodSignature                  = objc.RegisterName("invocationWithMethodSignature:")
	sel_setSelector                                    = objc.RegisterName("setSelector:")
	sel_setTarget                                      = objc.RegisterName("setTarget:")
	sel_setArgumentAtIndex                             = objc.RegisterName("setArgument:atIndex:")
	sel_getReturnValue                                 = objc.RegisterName("getReturnValue:")
	sel_invoke                                         = objc.RegisterName("invoke")
	sel_invokeWithTarget                               = objc.RegisterName("invokeWithTarget:")
	sel_instanceMethodSignatureForSelector             = objc.RegisterName("instanceMethodSignatureForSelector:")
	sel_signatureWithObjCTypes                         = objc.RegisterName("signatureWithObjCTypes:")
	sel_initWithUTF8String                             = objc.RegisterName("initWithUTF8String:")
	sel_UTF8String                                     = objc.RegisterName("UTF8String")
	sel_length                                         = objc.RegisterName("length")
	sel_processInfo                                    = objc.RegisterName("processInfo")
	sel_isOperatingSystemAtLeastVersion                = objc.RegisterName("isOperatingSystemAtLeastVersion:")
	sel_frame                                          = objc.RegisterName("frame")
	sel_contentView                                    = objc.RegisterName("contentView")
	sel_setBackgroundColor                             = objc.RegisterName("setBackgroundColor:")
	sel_colorWithSRGBRedGreenBlueAlpha                 = objc.RegisterName("colorWithSRGBRed:green:blue:alpha:")
	sel_setFrameSize                                   = objc.RegisterName("setFrameSize:")
	sel_object                                         = objc.RegisterName("object")
	sel_styleMask                                      = objc.RegisterName("styleMask")
	sel_setStyleMask                                   = objc.RegisterName("setStyleMask:")
	sel_mainScreen                                     = objc.RegisterName("mainScreen")
	sel_screen                                         = objc.RegisterName("screen")
	sel_isVisible                                      = objc.RegisterName("isVisible")
	sel_deviceDescription                              = objc.RegisterName("deviceDescription")
	sel_objectForKey                                   = objc.RegisterName("objectForKey:")
	sel_unsignedIntValue                               = objc.RegisterName("unsignedIntValue")
	sel_detachNewThreadSelector_toTarget_withObject    = objc.RegisterName("detachNewThreadSelector:toTarget:withObject:")
	sel_sharedApplication                              = objc.RegisterName("sharedApplication")
	sel_setDelegate                                    = objc.RegisterName("setDelegate:")
	sel_screens                                        = objc.RegisterName("screens")
	sel_objectAtIndex                                  = objc.RegisterName("objectAtIndex:")
	sel_count                                          = objc.RegisterName("count")
	sel_respondsToSelector                             = objc.RegisterName("respondsToSelector:")
	sel_performSelector                                = objc.RegisterName("performSelector:")
	sel_IBeamCursor                                    = objc.RegisterName("IBeamCursor")
	sel_crosshairCursor                                = objc.RegisterName("crosshairCursor")
	sel_pointingHandCursor                             = objc.RegisterName("pointingHandCursor")
	sel_convertRectToBacking                           = objc.RegisterName("convertRectToBacking:")
	sel_nextEventMatchingMask_untilDate_inMode_dequeue = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	sel_distantPast                                    = objc.RegisterName("distantPast")
	sel_sendEvent                                      = objc.RegisterName("sendEvent:")
	sel_setTitle                                       = objc.RegisterName("setTitle:")
	sel_setMiniwindowTitle                             = objc.RegisterName("setMiniwindowTitle:")
	sel_orderFront                                     = objc.RegisterName("orderFront:")
	sel_activateIgnoringOtherApps                      = objc.RegisterName("activateIgnoringOtherApps:")
	sel_makeKeyAndOrderFront                           = objc.RegisterName("makeKeyAndOrderFront:")
	sel_isKeyWindow                                    = objc.RegisterName("isKeyWindow")
	sel_isMiniaturized                                 = objc.RegisterName("isMiniaturized")
	sel_initWithContentRect_styleMask_backing_defer    = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	sel_mouseLocationOutsideOfEventStream              = objc.RegisterName("mouseLocationOutsideOfEventStream")
)

var NSDefaultRunLoopMode = *(*NSRunLoopMode)(unsafe.Pointer(purego.Dlsym(Cocoa, "NSDefaultRunLoopMode")))
var NSApp = *(*NSApplication)(unsafe.Pointer(purego.Dlsym(Cocoa, "NSApp")))

const NSWindowCollectionBehaviorFullScreenPrimary = 1 << 7

type NSBackingStoreType NSUInteger

const (
	NSBackingStoreBuffered NSBackingStoreType = 2
)

type NSWindowStyleMask NSUInteger

const (
	NSWindowStyleMaskBorderless     NSWindowStyleMask = 0
	NSWindowStyleMaskTitled         NSWindowStyleMask = 1 << 0
	NSWindowStyleMaskClosable       NSWindowStyleMask = 1 << 1
	NSWindowStyleMaskMiniaturizable NSWindowStyleMask = 1 << 2
	NSWindowStyleMaskResizable      NSWindowStyleMask = 1 << 3
	NSWindowStyleMaskFullScreen     NSWindowStyleMask = 1 << 14
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

type NSEventMask uint64

const NSUIntegerMax = math.MaxUint

const (
	NSEventMaskAny = NSUIntegerMax
)

type NSRunLoopMode NSString

func NSObject_retain(obj objc.ID) {
	obj.Send(sel_retain)
}

type NSError struct {
	objc.ID
}

type NSColor struct {
	objc.ID
}

func NSColor_colorWithSRGBRedGreenBlueAlpha(red, green, blue, alpha CGFloat) (color NSColor) {
	sig := NSMethodSignature_signatureWithObjCTypes("@@:ffff")
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_colorWithSRGBRedGreenBlueAlpha)
	inv.SetArgumentAtIndex(unsafe.Pointer(&red), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&green), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&blue), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&alpha), 5)
	inv.InvokeWithTarget(objc.ID(class_NSColor))
	inv.GetReturnValue(unsafe.Pointer(&color))
	return color
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

func (p NSProcessInfo) IsOperatingSystemAtLeastVersion(version NSOperatingSystemVersion) bool {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSProcessInfo), sel_isOperatingSystemAtLeastVersion)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(p.ID)
	inv.SetSelector(sel_isOperatingSystemAtLeastVersion)
	inv.SetArgumentAtIndex(unsafe.Pointer(&version), 2)
	inv.Invoke()
	var ret int
	inv.GetReturnValue(unsafe.Pointer(&ret))
	return ret != 0
}

type NSWindow struct {
	objc.ID
}

func (w NSWindow) InitWithContentRectStyleMaskBackingDefer(contentRect NSRect, style NSWindowStyleMask, backing NSBackingStoreType, flag bool) NSWindow {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSWindow), sel_initWithContentRect_styleMask_backing_defer)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(w.ID)
	inv.SetSelector(sel_initWithContentRect_styleMask_backing_defer)
	inv.SetArgumentAtIndex(unsafe.Pointer(&contentRect), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&style), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&backing), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&flag), 5)
	inv.Invoke()
	var ret objc.ID
	inv.GetReturnValue(unsafe.Pointer(&ret))
	return NSWindow{ret}
}

func (w NSWindow) IsMiniaturized() bool {
	return w.Send(sel_isMiniaturized) != 0
}

func (w NSWindow) MakeKeyAndOrderFront(sender objc.ID) {
	w.Send(sel_makeKeyAndOrderFront, sender)
}

func (w NSWindow) IsKeyWindow() bool {
	return w.Send(sel_isKeyWindow) != 0
}

func (w NSWindow) OrderFront(sender objc.ID) {
	w.Send(sel_orderFront, sender)
}

func (w NSWindow) SetTitle(t NSString) {
	w.Send(sel_setTitle, t.ID)
}

func (w NSWindow) SetDelegate(id objc.ID) {
	w.Send(sel_setDelegate, id)
}

func (w NSWindow) SetMiniwindowTitle(t NSString) {
	w.Send(sel_setMiniwindowTitle, t.ID)
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

func (w NSWindow) IsVisibile() bool {
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
	rect := NSRect{}
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
}
func (w NSWindow) MouseLocationOutsideOfEventStream() NSPoint {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSWindow), sel_mouseLocationOutsideOfEventStream)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(w.ID)
	inv.SetSelector(sel_mouseLocationOutsideOfEventStream)
	inv.Invoke()
	point := NSPoint{}
	inv.GetReturnValue(unsafe.Pointer(&point))
	return point
}

func (w NSWindow) ContentView() NSView {
	return NSView{w.Send(sel_contentView)}
}

type NSCursor struct {
	objc.ID
}

func NSCursor_IBeamCursor() NSCursor {
	return NSCursor{objc.ID(class_NSCursor).Send(sel_IBeamCursor)}
}

func NSCursor_crosshairCursor() NSCursor {
	return NSCursor{objc.ID(class_NSCursor).Send(sel_crosshairCursor)}
}

func NSCursor_pointingHandCursor() NSCursor {
	return NSCursor{objc.ID(class_NSCursor).Send(sel_pointingHandCursor)}
}

func NSCursor_respondsToSelector(sel objc.SEL) bool {
	return objc.ID(class_NSCursor).Send(sel_respondsToSelector, sel) != 0
}

func NSCursor_performSelector(sel objc.SEL) objc.ID {
	return objc.ID(class_NSCursor).Send(sel_performSelector, sel)
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
	rect := NSRect{}
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
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

func (s NSScreen) Frame() NSRect {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSScreen), sel_frame)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_frame)
	inv.InvokeWithTarget(s.ID)
	rect := NSRect{}
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
}

func (s NSScreen) ConvertRectToBacking(rect NSRect) NSRect {
	sig := NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(class_NSScreen), sel_convertRectToBacking)
	inv := NSInvocation_invocationWithMethodSignature(sig)
	inv.SetSelector(sel_convertRectToBacking)
	inv.InvokeWithTarget(s.ID)
	inv.GetReturnValue(unsafe.Pointer(&rect))
	return rect
}

type NSArray struct {
	objc.ID
}

func (a NSArray) Count() NSUInteger {
	return NSUInteger(a.Send(sel_count))
}

func (a NSArray) ObjectAtIndex(idx NSUInteger) objc.ID {
	return a.Send(sel_objectAtIndex, idx)
}

type NSThread struct {
	objc.ID
}

func NSThread_detachNewThreadSelectorToTargetWithObject(sel objc.SEL, target, argument objc.ID) {
	objc.ID(class_NSThread).Send(sel_detachNewThreadSelector_toTarget_withObject, sel, target, argument)
}

type NSApplication struct {
	objc.ID
}

func NSApplication_sharedApplication() NSApplication {
	return NSApplication{objc.ID(class_NSApplication).Send(sel_sharedApplication)}
}

func (a NSApplication) ActivateIgnoringOtherApps(b bool) {
	a.Send(sel_activateIgnoringOtherApps, b)
}

func (a NSApplication) SetDelegate(delegate objc.ID) {
	a.Send(sel_setDelegate, delegate)
}

func (a NSApplication) NextEventMatchingMaskUntilDateInModeDequeue(mask NSEventMask, expiration NSDate, mode NSRunLoopMode, dequeue bool) NSEvent {
	return NSEvent{a.Send(sel_nextEventMatchingMask_untilDate_inMode_dequeue, mask, expiration.ID, mode.ID, dequeue)}
}

func (a NSApplication) SendEvent(event NSEvent) {
	a.Send(sel_sendEvent, event.ID)
}

type NSDate struct {
	objc.ID
}

func NSDate_distantPast() NSDate {
	return NSDate{objc.ID(class_NSDate).Send(sel_distantPast)}
}

type NSEvent struct {
	objc.ID
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

func NSScreen_screens() NSArray {
	return NSArray{objc.ID(class_NSScreen).Send(sel_screens)}
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

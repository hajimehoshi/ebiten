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
	class_NSInvocation         = objc.GetClass("NSInvocation")
	class_NSMethodSignature    = objc.GetClass("NSMethodSignature")
	class_NSAutoreleasePool    = objc.GetClass("NSAutoreleasePool")
	class_NSString             = objc.GetClass("NSString")
	class_NSColor              = objc.GetClass("NSColor")
	class_NSScreen             = objc.GetClass("NSScreen")
	class_NSRunLoop            = objc.GetClass("NSRunLoop")
	class_NSMachPort           = objc.GetClass("NSMachPort")
	class_NSWorkspace          = objc.GetClass("NSWorkspace")
	class_NSNotificationCenter = objc.GetClass("NSNotificationCenter")
	class_NSOperationQueue     = objc.GetClass("NSOperationQueue")
)

var (
	sel_retain                         = objc.RegisterName("retain")
	sel_alloc                          = objc.RegisterName("alloc")
	sel_new                            = objc.RegisterName("new")
	sel_release                        = objc.RegisterName("release")
	sel_invocationWithMethodSignature  = objc.RegisterName("invocationWithMethodSignature:")
	sel_setSelector                    = objc.RegisterName("setSelector:")
	sel_setTarget                      = objc.RegisterName("setTarget:")
	sel_setArgumentAtIndex             = objc.RegisterName("setArgument:atIndex:")
	sel_getReturnValue                 = objc.RegisterName("getReturnValue:")
	sel_invoke                         = objc.RegisterName("invoke")
	sel_invokeWithTarget               = objc.RegisterName("invokeWithTarget:")
	sel_signatureWithObjCTypes         = objc.RegisterName("signatureWithObjCTypes:")
	sel_initWithUTF8String             = objc.RegisterName("initWithUTF8String:")
	sel_UTF8String                     = objc.RegisterName("UTF8String")
	sel_length                         = objc.RegisterName("length")
	sel_frame                          = objc.RegisterName("frame")
	sel_contentView                    = objc.RegisterName("contentView")
	sel_setBackgroundColor             = objc.RegisterName("setBackgroundColor:")
	sel_colorWithSRGBRedGreenBlueAlpha = objc.RegisterName("colorWithSRGBRed:green:blue:alpha:")
	sel_setFrameSize                   = objc.RegisterName("setFrameSize:")
	sel_object                         = objc.RegisterName("object")
	sel_styleMask                      = objc.RegisterName("styleMask")
	sel_setStyleMask                   = objc.RegisterName("setStyleMask:")
	sel_mainScreen                     = objc.RegisterName("mainScreen")
	sel_screen                         = objc.RegisterName("screen")
	sel_isVisible                      = objc.RegisterName("isVisible")
	sel_deviceDescription              = objc.RegisterName("deviceDescription")
	sel_objectForKey                   = objc.RegisterName("objectForKey:")
	sel_unsignedIntValue               = objc.RegisterName("unsignedIntValue")
	sel_setLayer                       = objc.RegisterName("setLayer:")
	sel_setWantsLayer                  = objc.RegisterName("setWantsLayer:")
	sel_mainRunLoop                    = objc.RegisterName("mainRunLoop")
	sel_currentRunLoop                 = objc.RegisterName("currentRunLoop")
	sel_run                            = objc.RegisterName("run")
	sel_performBlock                   = objc.RegisterName("performBlock:")
	sel_port                           = objc.RegisterName("port")
	sel_addPort                        = objc.RegisterName("addPort:forMode:")
	sel_sharedWorkspace                = objc.RegisterName("sharedWorkspace")
	sel_notificationCenter             = objc.RegisterName("notificationCenter")
	sel_addObserver                    = objc.RegisterName("addObserver:selector:name:object:")
	sel_addObserverForName             = objc.RegisterName("addObserverForName:object:queue:usingBlock:")
	sel_mainQueue                      = objc.RegisterName("mainQueue")
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

type NSObject struct {
	objc.ID
}

func (n NSObject) Retain() {
	n.Send(sel_retain)
}

type NSError struct {
	objc.ID
}

type NSColor struct {
	objc.ID
}

func NSColor_colorWithSRGBRedGreenBlueAlpha(red, green, blue, alpha CGFloat) (color NSColor) {
	return NSColor{objc.ID(class_NSColor).Send(sel_colorWithSRGBRedGreenBlueAlpha, red, green, blue, alpha)}
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
	return objc.Send[NSRect](w.ID, sel_frame)
}

func (w NSWindow) ContentView() NSView {
	return NSView{w.Send(sel_contentView)}
}

type NSView struct {
	objc.ID
}

func (v NSView) SetFrameSize(size CGSize) {
	v.ID.Send(sel_setFrameSize, size)
}

func (v NSView) Frame() NSRect {
	return objc.Send[NSRect](v.ID, sel_frame)
}

func (v NSView) SetLayer(layer uintptr) {
	v.Send(sel_setLayer, layer)
}

func (v NSView) SetWantsLayer(wantsLayer bool) {
	v.Send(sel_setWantsLayer, wantsLayer)
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

type NSRunLoop struct {
	objc.ID
}

func NSRunLoop_mainRunLoop() NSRunLoop {
	return NSRunLoop{objc.ID(class_NSRunLoop).Send(sel_mainRunLoop)}
}

func NSRunLoop_currentRunLoop() NSRunLoop {
	return NSRunLoop{objc.ID(class_NSRunLoop).Send(sel_currentRunLoop)}
}

func (r NSRunLoop) AddPort(port NSMachPort, mode NSRunLoopMode) {
	r.Send(sel_addPort, port.ID, mode)
}

func (r NSRunLoop) Run() {
	r.Send(sel_run)
}

func (r NSRunLoop) PerformBlock(block objc.Block) {
	r.Send(sel_performBlock, block)
}

type NSRunLoopMode NSString

var (
	NSRunLoopCommonModes = NSRunLoopMode(NSString_alloc().InitWithUTF8String("kCFRunLoopCommonModes"))
	NSDefaultRunLoopMode = NSRunLoopMode(NSString_alloc().InitWithUTF8String("kCFRunLoopDefaultMode"))
)

type NSMachPort struct {
	objc.ID
}

func NSMachPort_port() NSMachPort {
	return NSMachPort{objc.ID(class_NSMachPort).Send(sel_port)}
}

type NSWorkspace struct {
	objc.ID
}

func NSWorkspace_sharedWorkspace() NSWorkspace {
	return NSWorkspace{objc.ID(class_NSWorkspace).Send(sel_sharedWorkspace)}
}

func (w NSWorkspace) NotificationCenter() NSNotificationCenter {
	return NSNotificationCenter{w.Send(sel_notificationCenter)}
}

var (
	NSWorkspaceDidWakeNotification        = NSString_alloc().InitWithUTF8String("NSWorkspaceDidWakeNotification")
	NSWorkspaceScreensDidWakeNotification = NSString_alloc().InitWithUTF8String("NSWorkspaceScreensDidWakeNotification")
)

type NSNotificationCenter struct {
	objc.ID
}

func (n NSNotificationCenter) AddObserverForName(name NSString, object objc.ID, queue NSOperationQueue, usingBlock objc.Block) objc.ID {
	return n.Send(sel_addObserverForName, name.ID, object, queue.ID, usingBlock)
}

type NSOperationQueue struct {
	objc.ID
}

func NSOperationQueue_mainQueue() NSOperationQueue {
	return NSOperationQueue{objc.ID(class_NSOperationQueue).Send(sel_mainQueue)}
}

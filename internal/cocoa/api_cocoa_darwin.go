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
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	classNSInvocation         objc.Class
	classNSMethodSignature    objc.Class
	classNSAutoreleasePool    objc.Class
	classNSString             objc.Class
	classNSColor              objc.Class
	classNSScreen             objc.Class
	classNSRunLoop            objc.Class
	classNSMachPort           objc.Class
	classNSWorkspace          objc.Class
	classNSNotificationCenter objc.Class
	classNSOperationQueue     objc.Class
)

func init() {
	// Load Foundation and AppKit frameworks to ensure ObjC classes are available.
	// With CGO_ENABLED=0, frameworks are not loaded automatically by the linker,
	// so objc.GetClass would return 0 for classes like NSString.
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("cocoa: failed to dlopen Foundation: %w", err))
	}

	// AppKit may not be available (e.g. on iOS), so ignore errors.
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("cocoa: failed to dlopen AppKit: %w", err))
	}

	classNSInvocation = objc.GetClass("NSInvocation")
	classNSMethodSignature = objc.GetClass("NSMethodSignature")
	classNSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	classNSString = objc.GetClass("NSString")
	classNSColor = objc.GetClass("NSColor")
	classNSScreen = objc.GetClass("NSScreen")
	classNSRunLoop = objc.GetClass("NSRunLoop")
	classNSMachPort = objc.GetClass("NSMachPort")
	classNSWorkspace = objc.GetClass("NSWorkspace")
	classNSNotificationCenter = objc.GetClass("NSNotificationCenter")
	classNSOperationQueue = objc.GetClass("NSOperationQueue")

	NSRunLoopCommonModes = NSRunLoopMode(NSString_alloc().InitWithUTF8String("kCFRunLoopCommonModes"))
	NSDefaultRunLoopMode = NSRunLoopMode(NSString_alloc().InitWithUTF8String("kCFRunLoopDefaultMode"))

	NSWorkspaceDidWakeNotification = NSString_alloc().InitWithUTF8String("NSWorkspaceDidWakeNotification")
	NSWorkspaceScreensDidWakeNotification = NSString_alloc().InitWithUTF8String("NSWorkspaceScreensDidWakeNotification")
}

var (
	selRetain                         = objc.RegisterName("retain")
	selAlloc                          = objc.RegisterName("alloc")
	selNew                            = objc.RegisterName("new")
	selRelease                        = objc.RegisterName("release")
	selInvocationWithMethodSignature  = objc.RegisterName("invocationWithMethodSignature:")
	selSetSelector                    = objc.RegisterName("setSelector:")
	selSetTarget                      = objc.RegisterName("setTarget:")
	selSetArgumentAtIndex             = objc.RegisterName("setArgument:atIndex:")
	selGetReturnValue                 = objc.RegisterName("getReturnValue:")
	selInvoke                         = objc.RegisterName("invoke")
	selInvokeWithTarget               = objc.RegisterName("invokeWithTarget:")
	selSignatureWithObjCTypes         = objc.RegisterName("signatureWithObjCTypes:")
	selInitWithUTF8String             = objc.RegisterName("initWithUTF8String:")
	selUTF8String                     = objc.RegisterName("UTF8String")
	selLength                         = objc.RegisterName("length")
	selLengthOfBytesUsingEncoding     = objc.RegisterName("lengthOfBytesUsingEncoding:")
	selFrame                          = objc.RegisterName("frame")
	selContentView                    = objc.RegisterName("contentView")
	selSetBackgroundColor             = objc.RegisterName("setBackgroundColor:")
	selColorWithSRGBRedGreenBlueAlpha = objc.RegisterName("colorWithSRGBRed:green:blue:alpha:")
	selSetFrameSize                   = objc.RegisterName("setFrameSize:")
	selObject                         = objc.RegisterName("object")
	selStyleMask                      = objc.RegisterName("styleMask")
	selSetStyleMask                   = objc.RegisterName("setStyleMask:")
	selMainScreen                     = objc.RegisterName("mainScreen")
	selScreen                         = objc.RegisterName("screen")
	selIsVisible                      = objc.RegisterName("isVisible")
	selDeviceDescription              = objc.RegisterName("deviceDescription")
	selObjectForKey                   = objc.RegisterName("objectForKey:")
	selUnsignedIntValue               = objc.RegisterName("unsignedIntValue")
	selSetLayer                       = objc.RegisterName("setLayer:")
	selSetWantsLayer                  = objc.RegisterName("setWantsLayer:")
	selMainRunLoop                    = objc.RegisterName("mainRunLoop")
	selCurrentRunLoop                 = objc.RegisterName("currentRunLoop")
	selRun                            = objc.RegisterName("run")
	selPerformBlock                   = objc.RegisterName("performBlock:")
	selPort                           = objc.RegisterName("port")
	selAddPort                        = objc.RegisterName("addPort:forMode:")
	selSharedWorkspace                = objc.RegisterName("sharedWorkspace")
	selNotificationCenter             = objc.RegisterName("notificationCenter")
	selAddObserver                    = objc.RegisterName("addObserver:selector:name:object:")
	selAddObserverForName             = objc.RegisterName("addObserverForName:object:queue:usingBlock:")
	selMainQueue                      = objc.RegisterName("mainQueue")
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
	n.Send(selRetain)
}

type NSError struct {
	objc.ID
}

type NSColor struct {
	objc.ID
}

func NSColor_colorWithSRGBRedGreenBlueAlpha(red, green, blue, alpha CGFloat) (color NSColor) {
	return NSColor{objc.ID(classNSColor).Send(selColorWithSRGBRedGreenBlueAlpha, red, green, blue, alpha)}
}

type NSWindow struct {
	objc.ID
}

func (w NSWindow) StyleMask() NSUInteger {
	return NSUInteger(w.Send(selStyleMask))
}

func (w NSWindow) SetStyleMask(styleMask NSUInteger) {
	w.Send(selSetStyleMask, styleMask)
}

func (w NSWindow) SetBackgroundColor(color NSColor) {
	w.Send(selSetBackgroundColor, color.ID)
}

func (w NSWindow) IsVisible() bool {
	return w.Send(selIsVisible) != 0
}

func (w NSWindow) Screen() NSScreen {
	return NSScreen{w.Send(selScreen)}
}

func (w NSWindow) Frame() NSRect {
	return objc.Send[NSRect](w.ID, selFrame)
}

func (w NSWindow) ContentView() NSView {
	return NSView{w.Send(selContentView)}
}

type NSView struct {
	objc.ID
}

func (v NSView) SetFrameSize(size CGSize) {
	v.ID.Send(selSetFrameSize, size)
}

func (v NSView) Frame() NSRect {
	return objc.Send[NSRect](v.ID, selFrame)
}

func (v NSView) SetLayer(layer uintptr) {
	v.Send(selSetLayer, layer)
}

func (v NSView) SetWantsLayer(wantsLayer bool) {
	v.Send(selSetWantsLayer, wantsLayer)
}

// NSInvocation is being used to call functions that can't be called directly with purego.SyscallN.
// See the downsides of that function for what it cannot do.
type NSInvocation struct {
	objc.ID
}

func NSInvocation_invocationWithMethodSignature(sig NSMethodSignature) NSInvocation {
	return NSInvocation{objc.ID(classNSInvocation).Send(selInvocationWithMethodSignature, sig.ID)}
}

func (i NSInvocation) SetSelector(cmd objc.SEL) {
	i.Send(selSetSelector, cmd)
}

func (i NSInvocation) SetTarget(target objc.ID) {
	i.Send(selSetTarget, target)
}

func (i NSInvocation) SetArgumentAtIndex(arg unsafe.Pointer, idx int) {
	i.Send(selSetArgumentAtIndex, arg, idx)
}

func (i NSInvocation) GetReturnValue(ret unsafe.Pointer) {
	i.Send(selGetReturnValue, ret)
}

func (i NSInvocation) Invoke() {
	i.Send(selInvoke)
}

func (i NSInvocation) InvokeWithTarget(target objc.ID) {
	i.Send(selInvokeWithTarget, target)
}

type NSMethodSignature struct {
	objc.ID
}

// NSMethodSignature_signatureWithObjCTypes takes a string that represents the type signature of a method.
// It follows the encoding specified in the Apple Docs.
//
// [Apple Docs]: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/ObjCRuntimeGuide/Articles/ocrtTypeEncodings.html#//apple_ref/doc/uid/TP40008048-CH100
func NSMethodSignature_signatureWithObjCTypes(types string) NSMethodSignature {
	return NSMethodSignature{objc.ID(classNSMethodSignature).Send(selSignatureWithObjCTypes, types)}
}

type NSAutoreleasePool struct {
	objc.ID
}

func NSAutoreleasePool_new() NSAutoreleasePool {
	return NSAutoreleasePool{objc.ID(classNSAutoreleasePool).Send(selNew)}
}

func (p NSAutoreleasePool) Release() {
	p.Send(selRelease)
}

type NSString struct {
	objc.ID
}

func NSString_alloc() NSString {
	return NSString{objc.ID(classNSString).Send(selAlloc)}
}

func (s NSString) InitWithUTF8String(utf8 string) NSString {
	return NSString{s.Send(selInitWithUTF8String, utf8)}
}

func (s NSString) String() string {
	// Use lengthOfBytesUsingEncoding: with NSUTF8StringEncoding (4) to get the
	// correct UTF-8 byte count. NSString.length returns UTF-16 code units which
	// differs from UTF-8 byte count for non-ASCII characters.
	length := s.Send(selLengthOfBytesUsingEncoding, 4)
	return string(unsafe.Slice((*byte)(unsafe.Pointer(s.Send(selUTF8String))), length))
}

type NSNotification struct {
	objc.ID
}

func (n NSNotification) Object() objc.ID {
	return n.Send(selObject)
}

type NSScreen struct {
	objc.ID
}

func NSScreen_mainScreen() NSScreen {
	return NSScreen{objc.ID(classNSScreen).Send(selMainScreen)}
}

func (s NSScreen) DeviceDescription() NSDictionary {
	return NSDictionary{s.Send(selDeviceDescription)}
}

type NSDictionary struct {
	objc.ID
}

func (d NSDictionary) ObjectForKey(object objc.ID) objc.ID {
	return d.Send(selObjectForKey, object)
}

type NSNumber struct {
	objc.ID
}

func (n NSNumber) UnsignedIntValue() uint {
	return uint(n.Send(selUnsignedIntValue))
}

type NSRunLoop struct {
	objc.ID
}

func NSRunLoop_mainRunLoop() NSRunLoop {
	return NSRunLoop{objc.ID(classNSRunLoop).Send(selMainRunLoop)}
}

func NSRunLoop_currentRunLoop() NSRunLoop {
	return NSRunLoop{objc.ID(classNSRunLoop).Send(selCurrentRunLoop)}
}

func (r NSRunLoop) AddPort(port NSMachPort, mode NSRunLoopMode) {
	r.Send(selAddPort, port.ID, mode)
}

func (r NSRunLoop) Run() {
	r.Send(selRun)
}

func (r NSRunLoop) PerformBlock(block objc.Block) {
	r.Send(selPerformBlock, block)
}

type NSRunLoopMode NSString

var (
	NSRunLoopCommonModes NSRunLoopMode
	NSDefaultRunLoopMode NSRunLoopMode
)

type NSMachPort struct {
	objc.ID
}

func NSMachPort_port() NSMachPort {
	return NSMachPort{objc.ID(classNSMachPort).Send(selPort)}
}

type NSWorkspace struct {
	objc.ID
}

func NSWorkspace_sharedWorkspace() NSWorkspace {
	return NSWorkspace{objc.ID(classNSWorkspace).Send(selSharedWorkspace)}
}

func (w NSWorkspace) NotificationCenter() NSNotificationCenter {
	return NSNotificationCenter{w.Send(selNotificationCenter)}
}

var (
	NSWorkspaceDidWakeNotification        NSString
	NSWorkspaceScreensDidWakeNotification NSString
)

type NSNotificationCenter struct {
	objc.ID
}

func (n NSNotificationCenter) AddObserverForName(name NSString, object objc.ID, queue NSOperationQueue, usingBlock objc.Block) objc.ID {
	return n.Send(selAddObserverForName, name.ID, object, queue.ID, usingBlock)
}

type NSOperationQueue struct {
	objc.ID
}

func NSOperationQueue_mainQueue() NSOperationQueue {
	return NSOperationQueue{objc.ID(classNSOperationQueue).Send(selMainQueue)}
}

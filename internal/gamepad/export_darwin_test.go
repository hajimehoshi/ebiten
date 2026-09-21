// Copyright 2026 The Ebitengine Authors
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

//go:build !ios

package gamepad

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

type (
	Gamepads     = gamepads
	Controller   = objc.ID
	HIDDeviceRef = _IOHIDDeviceRef
)

const (
	ControllerButtonA             = kControllerButtonA
	ControllerButtonB             = kControllerButtonB
	ControllerButtonX             = kControllerButtonX
	ControllerButtonY             = kControllerButtonY
	ControllerButtonBack          = kControllerButtonBack
	ControllerButtonStart         = kControllerButtonStart
	ControllerButtonLeftShoulder  = kControllerButtonLeftShoulder
	ControllerButtonRightShoulder = kControllerButtonRightShoulder
)

var (
	sel_controllerWithMicroGamepad    = objc.RegisterName("controllerWithMicroGamepad")
	sel_controllerWithExtendedGamepad = objc.RegisterName("controllerWithExtendedGamepad")
	sel_initWithInt                   = objc.RegisterName("initWithInt:")
	sel_setObject_forKey              = objc.RegisterName("setObject:forKey:")
)

// gcControllerClassForTest is the real GCController class, saved while InstallTestHIDBoundary has
// class_GCController replaced with the stub.
var gcControllerClassForTest objc.Class

// gcClaimsDeviceForTest stands in for +[GCController supportsHIDDevice:] while the boundary is
// installed.
var gcClaimsDeviceForTest func(HIDDeviceRef) bool

var testMicroControllerClass objc.Class

// realGCControllerClass returns the GCController class, also while the boundary is installed.
func realGCControllerClass() objc.Class {
	if gcControllerClassForTest != 0 {
		return gcControllerClassForTest
	}
	return class_GCController
}

// InitializeCF loads the CoreFoundation symbols the test harness uses.
func InitializeCF() error {
	return initializeCF()
}

// SupportsTestControllers reports whether +[GCController controllerWithMicroGamepad] and
// +[GCController controllerWithExtendedGamepad], the class methods [NewMicroControllerForTest] and
// [NewExtendedControllerForTest] build their controllers with, exist on this macOS.
func SupportsTestControllers() bool {
	class := realGCControllerClass()
	if class == 0 {
		return false
	}
	return objc.ID(class).Send(sel_respondsToSelector, sel_controllerWithMicroGamepad) != 0 &&
		objc.ID(class).Send(sel_respondsToSelector, sel_controllerWithExtendedGamepad) != 0
}

// InstallTestHIDBoundary replaces the hardware boundary of the two darwin backends: claims decides
// which HID devices the GameController framework owns, HID device properties are read from the
// dictionary [NewHIDDeviceForTest] builds, and a HID device reports no elements. Everything else —
// CF objects, GC profiles, callbacks, registration, removal and reference ownership — stays
// production code. The returned function restores what was swapped.
func InstallTestHIDBoundary(claims func(HIDDeviceRef) bool) (func(), error) {
	class := objc.GetClass("EbitengineTestHIDSupport")
	if class == 0 {
		var err error
		class, err = objc.RegisterClass("EbitengineTestHIDSupport", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
			Cmd: sel_supportsHIDDevice,
			Fn: func(_ objc.ID, _ objc.SEL, device uintptr) bool {
				return gcClaimsDeviceForTest(HIDDeviceRef(device))
			},
		}})
		if err != nil {
			return nil, err
		}
	}
	support := objc.ID(class).Send(sel_alloc).Send(sel_init)

	oldClass, oldGC, oldIOKit := class_GCController, theGCGamepads, theIOKitGamepads
	oldProperty, oldElements := _IOHIDDeviceGetProperty, _IOHIDDeviceCopyMatchingElements
	oldHaptics := coreHapticsAvailable

	gcControllerClassForTest = oldClass
	gcClaimsDeviceForTest = claims
	// class_GCController is only ever used as a message receiver, so the stub instance can stand in
	// for the class.
	class_GCController = objc.Class(support)
	_IOHIDDeviceGetProperty = func(device _IOHIDDeviceRef, key _CFStringRef) _CFTypeRef {
		return _CFTypeRef(objc.ID(device).Send(sel_objectForKeyedSubscript, key))
	}
	_IOHIDDeviceCopyMatchingElements = func(_IOHIDDeviceRef, _CFDictionaryRef, _IOOptionBits) _CFArrayRef {
		return _CFArrayRef(objc.ID(objc.GetClass("NSArray")).Send(sel_alloc).Send(sel_init))
	}
	// The test does not read rumble, and a virtual extended controller would otherwise start a real
	// CoreHaptics engine.
	coreHapticsAvailable = false

	return func() {
		class_GCController, theGCGamepads, theIOKitGamepads = oldClass, oldGC, oldIOKit
		_IOHIDDeviceGetProperty, _IOHIDDeviceCopyMatchingElements = oldProperty, oldElements
		coreHapticsAvailable = oldHaptics
		gcControllerClassForTest = 0
		gcClaimsDeviceForTest = nil
		support.Send(sel_release)
	}, nil
}

// NewDarwinGamepadsForTest returns the composite darwin backend over a fresh GameController and
// IOKit backend, and installs those as the running backends the notifications and device callbacks
// report to.
func NewDarwinGamepadsForTest() *nativeGamepadsDarwin {
	gc, iokit := &nativeGamepadsGC{}, &nativeGamepadsIOKit{}
	theGCGamepads, theIOKitGamepads = gc, iokit
	return &nativeGamepadsDarwin{gc: gc, iokit: iokit}
}

// Update applies the queued controller and device events to gamepads.
func (g *nativeGamepadsDarwin) Update(gamepads *gamepads) error {
	return g.update(gamepads)
}

// NewMicroControllerForTest returns an autoreleased GCController with only a micro gamepad profile.
func NewMicroControllerForTest() Controller {
	return objc.ID(realGCControllerClass()).Send(sel_controllerWithMicroGamepad)
}

// NewPressedButtonMicroControllerForTest returns a controller with a micro gamepad profile and one
// pressed input in its physical input profile.
func NewPressedButtonMicroControllerForTest(input int) (Controller, error) {
	if testMicroControllerClass == 0 {
		var err error
		testMicroControllerClass, err = objc.RegisterClass("EbitengineFullButtonMicroController", objc.GetClass("NSObject"), nil, []objc.FieldDef{
			{Name: "microGamepad", Type: reflect.TypeFor[objc.ID](), Attribute: objc.ReadOnly},
			{Name: "physicalInputProfile", Type: reflect.TypeFor[objc.ID](), Attribute: objc.ReadOnly},
			{Name: "vendorName", Type: reflect.TypeFor[objc.ID](), Attribute: objc.ReadOnly},
		}, []objc.MethodDef{{
			Cmd: sel_extendedGamepad,
			Fn:  func(objc.ID, objc.SEL) objc.ID { return 0 },
		}})
		if err != nil {
			return 0, err
		}
	}

	buttonClass := objc.GetClass("EbitenginePressedButton")
	if buttonClass == 0 {
		var err error
		buttonClass, err = objc.RegisterClass("EbitenginePressedButton", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
			Cmd: sel_isPressed,
			Fn:  func(objc.ID, objc.SEL) bool { return true },
		}})
		if err != nil {
			return 0, err
		}
	}

	buttons := objc.ID(objc.GetClass("NSMutableDictionary")).Send(sel_alloc).Send(sel_init)
	buttons.Send(objc.RegisterName("autorelease"))
	name := map[int]string{
		kControllerButtonB:             "Button B",
		kControllerButtonY:             "Button Y",
		kControllerButtonBack:          "Button Options",
		kControllerButtonLeftShoulder:  "Left Shoulder",
		kControllerButtonRightShoulder: "Right Shoulder",
	}[input]
	key := objc.ID(objc.GetClass("NSString")).Send(sel_alloc).Send(sel_initWithUTF8String, name+"\x00")
	button := objc.ID(buttonClass).Send(sel_alloc).Send(sel_init)
	buttons.Send(sel_setObject_forKey, button, key)
	button.Send(sel_release)
	key.Send(sel_release)
	profile := objc.ID(objc.GetClass("EbitenginePhysicalInputProfile"))
	if profile == 0 {
		var err error
		class, err := objc.RegisterClass("EbitenginePhysicalInputProfile", objc.GetClass("NSObject"), nil, []objc.FieldDef{
			{Name: "buttons", Type: reflect.TypeFor[objc.ID](), Attribute: objc.ReadOnly},
		}, nil)
		if err != nil {
			return 0, err
		}
		profile = objc.ID(class)
	}
	p := profile.Send(sel_alloc).Send(sel_init)
	p.Send(objc.RegisterName("autorelease"))
	p.SetIvar(objc.Class(profile).InstanceVariable("buttons"), buttons)

	microController := NewMicroControllerForTest()
	c := objc.ID(testMicroControllerClass).Send(sel_alloc).Send(sel_init)
	c.SetIvar(testMicroControllerClass.InstanceVariable("microGamepad"), microController.Send(sel_microGamepad))
	c.SetIvar(testMicroControllerClass.InstanceVariable("physicalInputProfile"), p)
	c.SetIvar(testMicroControllerClass.InstanceVariable("vendorName"), microController.Send(sel_vendorName))
	return c.Send(objc.RegisterName("autorelease")), nil
}

// NewExtendedControllerForTest returns an autoreleased GCController with an extended gamepad profile.
func NewExtendedControllerForTest() Controller {
	return objc.ID(realGCControllerClass()).Send(sel_controllerWithExtendedGamepad)
}

// AddController queues controller for registration, as its connect notification does.
func AddController(controller Controller) {
	addController(controller)
}

// RemoveController queues controller for removal, as its disconnect notification does.
func RemoveController(controller Controller) {
	removeController(controller)
}

// HIDDeviceArrived reports device to the IOKit backend, as the HID manager's matching callback does.
func HIDDeviceArrived(device HIDDeviceRef) {
	ebitenGamepadMatchingCallback(nil, 0, nil, device)
}

// HIDDeviceRemoved reports device to the IOKit backend, as the HID manager's removal callback does.
func HIDDeviceRemoved(device HIDDeviceRef) {
	ebitenGamepadRemovalCallback(nil, 0, nil, device)
}

// NewHIDDeviceForTest returns a HID device carrying the given properties. The device is the
// dictionary the stubbed IOHIDDeviceGetProperty reads; release it with [ReleaseHIDDeviceForTest].
func NewHIDDeviceForTest(name string, vendor, product int) HIDDeviceRef {
	d := objc.ID(objc.GetClass("NSMutableDictionary")).Send(sel_alloc).Send(sel_init)
	for _, p := range []struct {
		key   []byte
		value objc.ID
	}{
		// The keys are the null-terminated ones the IOKit backend looks up.
		{kIOHIDProductKey, objc.ID(objc.GetClass("NSString")).Send(sel_alloc).Send(sel_initWithUTF8String, name+"\x00")},
		{kIOHIDVendorIDKey, objc.ID(objc.GetClass("NSNumber")).Send(sel_alloc).Send(sel_initWithInt, vendor)},
		{kIOHIDProductIDKey, objc.ID(objc.GetClass("NSNumber")).Send(sel_alloc).Send(sel_initWithInt, product)},
	} {
		k := objc.ID(objc.GetClass("NSString")).Send(sel_alloc).Send(sel_initWithUTF8String, string(p.key))
		d.Send(sel_setObject_forKey, p.value, k)
		k.Send(sel_release)
		p.value.Send(sel_release)
	}
	return HIDDeviceRef(d)
}

// ReleaseHIDDeviceForTest releases a device from [NewHIDDeviceForTest].
func ReleaseHIDDeviceForTest(device HIDDeviceRef) {
	_CFRelease(_CFTypeRef(device))
}

// HIDDeviceProductID returns device's ProductID, read through the property path the IOKit backend
// uses.
func HIDDeviceProductID(device HIDDeviceRef) int {
	var product uint32
	if prop := hidDeviceProperty(device, kIOHIDProductIDKey); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&product))
	}
	return int(product)
}

// The backend that registered a gamepad, as [RegisteredGamepad] reports it.
const (
	NativeGC  = "gc"
	NativeHID = "hid"
)

// RegisteredGamepad describes one gamepad in the gamepad list.
type RegisteredGamepad struct {
	Gamepad *Gamepad
	// Native is the backend that registered the gamepad: [NativeGC] or [NativeHID].
	Native string
	// Micro reports that the GameController backend reads the gamepad through its micro gamepad
	// profile.
	Micro bool
	Name  string
	// Device is the IOKit device of a [NativeHID] gamepad, and 0 otherwise.
	Device HIDDeviceRef
}

// AppendRegisteredGamepads appends a description of every gamepad registered in g.
func (g *gamepads) AppendRegisteredGamepads(rs []RegisteredGamepad) []RegisteredGamepad {
	g.m.Lock()
	defer g.m.Unlock()

	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		r := RegisteredGamepad{
			Gamepad: gp,
			Name:    gp.Name(),
		}
		switch n := gp.native.(type) {
		case *nativeGamepadGC:
			r.Native = NativeGC
			r.Micro = n.micro
		case *nativeGamepadHID:
			r.Native = NativeHID
			r.Device = n.device
		}
		rs = append(rs, r)
	}
	return rs
}

// CloseAll releases the native resources of every gamepad registered in g.
func (g *gamepads) CloseAll() {
	g.m.Lock()
	defer g.m.Unlock()

	for _, gp := range g.gamepads {
		if gp != nil {
			gp.close()
		}
	}
}

// UpdateForTest reads one report from the gamepad's device.
func (g *Gamepad) UpdateForTest(gamepads *gamepads) error {
	return g.update(gamepads)
}

// GCButtonMask returns the mask of the buttons the GameController backend reads from the gamepad.
func (g *Gamepad) GCButtonMask() uint32 {
	var mask uint32
	withNative(g, func(n *nativeGamepadGC) {
		mask = n.buttonMask
	})
	return mask
}

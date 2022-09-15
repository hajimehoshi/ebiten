// Copyright 2022 The Ebiten Authors
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

//go:build !ios && !nintendosdk
// +build !ios,!nintendosdk

package gamepad

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

// #cgo LDFLAGS: -framework CoreFoundation -framework IOKit
//
// #include <ForceFeedback/ForceFeedback.h>
// #include <IOKit/hid/IOHIDLib.h>
//
// static CFStringRef cfStringRefIOHIDVendorIDKey() {
//   return CFSTR(kIOHIDVendorIDKey);
// }
//
// static CFStringRef cfStringRefIOHIDProductIDKey() {
//   return CFSTR(kIOHIDProductIDKey);
// }
//
// static CFStringRef cfStringRefIOHIDVersionNumberKey() {
//   return CFSTR(kIOHIDVersionNumberKey);
// }
//
// static CFStringRef cfStringRefIOHIDProductKey() {
//   return CFSTR(kIOHIDProductKey);
// }
//
// static CFStringRef cfStringRefIOHIDDeviceUsagePageKey() {
//   return CFSTR(kIOHIDDeviceUsagePageKey);
// }
//
// static CFStringRef cfStringRefIOHIDDeviceUsageKey() {
//   return CFSTR(kIOHIDDeviceUsageKey);
// }
//
// void ebitenGamepadMatchingCallback(void *ctx, IOReturn res, void *sender, IOHIDDeviceRef device);
// void ebitenGamepadRemovalCallback(void *ctx, IOReturn res, void *sender, IOHIDDeviceRef device);
import "C"

type nativeGamepadsImpl struct {
	hidManager      C.IOHIDManagerRef
	devicesToAdd    []C.IOHIDDeviceRef
	devicesToRemove []C.IOHIDDeviceRef
	devicesM        sync.Mutex
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (g *nativeGamepadsImpl) init(gamepads *gamepads) error {
	var dicts []C.CFDictionaryRef

	page := C.kHIDPage_GenericDesktop
	for _, usage := range []uint{
		C.kHIDUsage_GD_Joystick,
		C.kHIDUsage_GD_GamePad,
		C.kHIDUsage_GD_MultiAxisController,
	} {
		pageRef := C.CFNumberCreate(C.kCFAllocatorDefault, C.kCFNumberIntType, unsafe.Pointer(&page))
		if pageRef == 0 {
			return errors.New("gamepad: CFNumberCreate returned nil")
		}
		defer C.CFRelease(C.CFTypeRef(pageRef))

		usageRef := C.CFNumberCreate(C.kCFAllocatorDefault, C.kCFNumberIntType, unsafe.Pointer(&usage))
		if usageRef == 0 {
			return errors.New("gamepad: CFNumberCreate returned nil")
		}
		defer C.CFRelease(C.CFTypeRef(usageRef))

		keys := []C.CFStringRef{
			C.cfStringRefIOHIDDeviceUsagePageKey(),
			C.cfStringRefIOHIDDeviceUsageKey(),
		}
		values := []C.CFNumberRef{
			pageRef,
			usageRef,
		}

		dict := C.CFDictionaryCreate(C.kCFAllocatorDefault,
			(*unsafe.Pointer)(unsafe.Pointer(&keys[0])),
			(*unsafe.Pointer)(unsafe.Pointer(&values[0])),
			C.CFIndex(len(keys)), &C.kCFTypeDictionaryKeyCallBacks, &C.kCFTypeDictionaryValueCallBacks)
		if dict == 0 {
			return errors.New("gamepad: CFDictionaryCreate returned nil")
		}
		defer C.CFRelease(C.CFTypeRef(dict))

		dicts = append(dicts, dict)
	}

	matching := C.CFArrayCreate(C.kCFAllocatorDefault,
		(*unsafe.Pointer)(unsafe.Pointer(&dicts[0])),
		C.CFIndex(len(dicts)), &C.kCFTypeArrayCallBacks)
	if matching == 0 {
		return errors.New("gamepad: CFArrayCreateMutable returned nil")
	}
	defer C.CFRelease(C.CFTypeRef(matching))

	g.hidManager = C.IOHIDManagerCreate(C.kCFAllocatorDefault, C.kIOHIDOptionsTypeNone)
	if C.IOHIDManagerOpen(g.hidManager, C.kIOHIDOptionsTypeNone) != C.kIOReturnSuccess {
		return errors.New("gamepad: IOHIDManagerOpen failed")
	}

	C.IOHIDManagerSetDeviceMatchingMultiple(g.hidManager, matching)
	C.IOHIDManagerRegisterDeviceMatchingCallback(g.hidManager, C.IOHIDDeviceCallback(C.ebitenGamepadMatchingCallback), nil)
	C.IOHIDManagerRegisterDeviceRemovalCallback(g.hidManager, C.IOHIDDeviceCallback(C.ebitenGamepadRemovalCallback), nil)

	C.IOHIDManagerScheduleWithRunLoop(g.hidManager, C.CFRunLoopGetMain(), C.kCFRunLoopDefaultMode)

	// Execute the run loop once in order to register any initially-attached gamepads.
	C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, 0, 0 /* false */)

	return nil
}

//export ebitenGamepadMatchingCallback
func ebitenGamepadMatchingCallback(ctx unsafe.Pointer, res C.IOReturn, sender unsafe.Pointer, device C.IOHIDDeviceRef) {
	n := theGamepads.native.(*nativeGamepadsImpl)
	n.devicesM.Lock()
	defer n.devicesM.Unlock()
	n.devicesToAdd = append(n.devicesToAdd, device)
}

//export ebitenGamepadRemovalCallback
func ebitenGamepadRemovalCallback(ctx unsafe.Pointer, res C.IOReturn, sender unsafe.Pointer, device C.IOHIDDeviceRef) {
	n := theGamepads.native.(*nativeGamepadsImpl)
	n.devicesM.Lock()
	defer n.devicesM.Unlock()
	n.devicesToRemove = append(n.devicesToRemove, device)
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	n := theGamepads.native.(*nativeGamepadsImpl)
	n.devicesM.Lock()
	defer n.devicesM.Unlock()

	for _, device := range g.devicesToAdd {
		g.addDevice(device, gamepads)
	}
	for _, device := range g.devicesToRemove {
		gamepads.remove(func(g *Gamepad) bool {
			return g.native.(*nativeGamepadImpl).device == device
		})
	}
	g.devicesToAdd = g.devicesToAdd[:0]
	g.devicesToRemove = g.devicesToRemove[:0]
	return nil
}

func (g *nativeGamepadsImpl) addDevice(device C.IOHIDDeviceRef, gamepads *gamepads) {
	if gamepads.find(func(g *Gamepad) bool {
		return g.native.(*nativeGamepadImpl).device == device
	}) != nil {
		return
	}

	name := "Unknown"
	if prop := C.IOHIDDeviceGetProperty(device, C.cfStringRefIOHIDProductKey()); prop != 0 {
		var cstr [256]C.char
		C.CFStringGetCString(C.CFStringRef(prop), &cstr[0], C.CFIndex(len(cstr)), C.kCFStringEncodingUTF8)
		name = C.GoString(&cstr[0])
	}

	var vendor uint32
	if prop := C.IOHIDDeviceGetProperty(device, C.cfStringRefIOHIDVendorIDKey()); prop != 0 {
		C.CFNumberGetValue(C.CFNumberRef(prop), C.kCFNumberSInt32Type, unsafe.Pointer(&vendor))
	}

	var product uint32
	if prop := C.IOHIDDeviceGetProperty(device, C.cfStringRefIOHIDProductIDKey()); prop != 0 {
		C.CFNumberGetValue(C.CFNumberRef(prop), C.kCFNumberSInt32Type, unsafe.Pointer(&product))
	}

	var version uint32
	if prop := C.IOHIDDeviceGetProperty(device, C.cfStringRefIOHIDVersionNumberKey()); prop != 0 {
		C.CFNumberGetValue(C.CFNumberRef(prop), C.kCFNumberSInt32Type, unsafe.Pointer(&version))
	}

	var sdlID string
	if vendor != 0 && product != 0 {
		sdlID = fmt.Sprintf("03000000%02x%02x0000%02x%02x0000%02x%02x0000",
			byte(vendor), byte(vendor>>8),
			byte(product), byte(product>>8),
			byte(version), byte(version>>8))
	} else {
		bs := []byte(name)
		if len(bs) < 12 {
			bs = append(bs, make([]byte, 12-len(bs))...)
		}
		sdlID = fmt.Sprintf("05000000%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
			bs[0], bs[1], bs[2], bs[3], bs[4], bs[5], bs[6], bs[7], bs[8], bs[9], bs[10], bs[11])
	}

	elements := C.IOHIDDeviceCopyMatchingElements(device, 0, C.kIOHIDOptionsTypeNone)
	defer C.CFRelease(C.CFTypeRef(elements))

	n := &nativeGamepadImpl{
		device: device,
	}
	gp := gamepads.add(name, sdlID)
	gp.native = n

	for i := C.CFIndex(0); i < C.CFArrayGetCount(elements); i++ {
		native := (C.IOHIDElementRef)(C.CFArrayGetValueAtIndex(elements, i))
		if C.CFGetTypeID(C.CFTypeRef(native)) != C.IOHIDElementGetTypeID() {
			continue
		}

		typ := C.IOHIDElementGetType(native)
		if typ != C.kIOHIDElementTypeInput_Axis &&
			typ != C.kIOHIDElementTypeInput_Button &&
			typ != C.kIOHIDElementTypeInput_Misc {
			continue
		}

		usage := C.IOHIDElementGetUsage(native)
		page := C.IOHIDElementGetUsagePage(native)

		switch page {
		case C.kHIDPage_GenericDesktop:
			switch usage {
			case C.kHIDUsage_GD_X, C.kHIDUsage_GD_Y, C.kHIDUsage_GD_Z,
				C.kHIDUsage_GD_Rx, C.kHIDUsage_GD_Ry, C.kHIDUsage_GD_Rz,
				C.kHIDUsage_GD_Slider, C.kHIDUsage_GD_Dial, C.kHIDUsage_GD_Wheel:
				n.axes = append(n.axes, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.axes),
					minimum: int(C.IOHIDElementGetLogicalMin(native)),
					maximum: int(C.IOHIDElementGetLogicalMax(native)),
				})
			case C.kHIDUsage_GD_Hatswitch:
				n.hats = append(n.hats, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.hats),
					minimum: int(C.IOHIDElementGetLogicalMin(native)),
					maximum: int(C.IOHIDElementGetLogicalMax(native)),
				})
			case C.kHIDUsage_GD_DPadUp, C.kHIDUsage_GD_DPadRight, C.kHIDUsage_GD_DPadDown, C.kHIDUsage_GD_DPadLeft,
				C.kHIDUsage_GD_SystemMainMenu, C.kHIDUsage_GD_Select, C.kHIDUsage_GD_Start:
				n.buttons = append(n.buttons, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.buttons),
					minimum: int(C.IOHIDElementGetLogicalMin(native)),
					maximum: int(C.IOHIDElementGetLogicalMax(native)),
				})
			}
		case C.kHIDPage_Simulation:
			switch usage {
			case C.kHIDUsage_Sim_Accelerator, C.kHIDUsage_Sim_Brake, C.kHIDUsage_Sim_Throttle, C.kHIDUsage_Sim_Rudder, C.kHIDUsage_Sim_Steering:
				n.axes = append(n.axes, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.axes),
					minimum: int(C.IOHIDElementGetLogicalMin(native)),
					maximum: int(C.IOHIDElementGetLogicalMax(native)),
				})
			}
		case C.kHIDPage_Button, C.kHIDPage_Consumer:
			n.buttons = append(n.buttons, element{
				native:  native,
				usage:   int(usage),
				index:   len(n.buttons),
				minimum: int(C.IOHIDElementGetLogicalMin(native)),
				maximum: int(C.IOHIDElementGetLogicalMax(native)),
			})
		}
	}

	sort.Stable(n.axes)
	sort.Stable(n.buttons)
	sort.Stable(n.hats)
}

type element struct {
	native  C.IOHIDElementRef
	usage   int
	index   int
	minimum int
	maximum int
}

type elements []element

func (e elements) Len() int {
	return len(e)
}

func (e elements) Less(i, j int) bool {
	if e[i].usage != e[j].usage {
		return e[i].usage < e[j].usage
	}
	if e[i].index != e[j].index {
		return e[i].index < e[j].index
	}
	return false
}

func (e elements) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type nativeGamepadImpl struct {
	device  C.IOHIDDeviceRef
	axes    elements
	buttons elements
	hats    elements

	axisValues   []float64
	buttonValues []bool
	hatValues    []int
}

func (g *nativeGamepadImpl) elementValue(e *element) int {
	var valueRef C.IOHIDValueRef
	if C.IOHIDDeviceGetValue(g.device, e.native, &valueRef) == C.kIOReturnSuccess {
		return int(C.IOHIDValueGetIntegerValue(valueRef))
	}

	return 0
}

func (g *nativeGamepadImpl) update(gamepads *gamepads) error {
	if cap(g.axisValues) < len(g.axes) {
		g.axisValues = make([]float64, len(g.axes))
	}
	g.axisValues = g.axisValues[:len(g.axes)]

	if cap(g.buttonValues) < len(g.buttons) {
		g.buttonValues = make([]bool, len(g.buttons))
	}
	g.buttonValues = g.buttonValues[:len(g.buttons)]

	if cap(g.hatValues) < len(g.hats) {
		g.hatValues = make([]int, len(g.hats))
	}
	g.hatValues = g.hatValues[:len(g.hats)]

	for i, a := range g.axes {
		raw := g.elementValue(&a)
		if raw < a.minimum {
			a.minimum = raw
		}
		if raw > a.maximum {
			a.maximum = raw
		}
		var value float64
		if size := a.maximum - a.minimum; size != 0 {
			value = 2*float64(raw-a.minimum)/float64(size) - 1
		}
		g.axisValues[i] = value
	}

	for i, b := range g.buttons {
		g.buttonValues[i] = (g.elementValue(&b) - b.minimum) > 0
	}

	hatStates := []int{
		hatUp,
		hatRightUp,
		hatRight,
		hatRightDown,
		hatDown,
		hatLeftDown,
		hatLeft,
		hatLeftUp,
	}
	for i, h := range g.hats {
		if state := g.elementValue(&h) - h.minimum; state < 0 || state >= len(hatStates) {
			g.hatValues[i] = hatCentered
		} else {
			g.hatValues[i] = hatStates[state]
		}
	}

	return nil
}

func (g *nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return false
}

func (g *nativeGamepadImpl) isStandardAxisAvailableInOwnMapping(axis gamepaddb.StandardAxis) bool {
	return false
}

func (g *nativeGamepadImpl) isStandardButtonAvailableInOwnMapping(button gamepaddb.StandardButton) bool {
	return false
}

func (g *nativeGamepadImpl) axisCount() int {
	return len(g.axisValues)
}

func (g *nativeGamepadImpl) buttonCount() int {
	return len(g.buttonValues)
}

func (g *nativeGamepadImpl) hatCount() int {
	return len(g.hatValues)
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axisValues) {
		return 0
	}
	return g.axisValues[axis]
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	panic("gamepad: buttonValue is not implemented")
}

func (g *nativeGamepadImpl) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttonValues) {
		return false
	}
	return g.buttonValues[button]
}

func (g *nativeGamepadImpl) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hatValues) {
		return hatCentered
	}
	return g.hatValues[hat]
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}

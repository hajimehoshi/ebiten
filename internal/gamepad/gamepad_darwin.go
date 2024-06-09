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

//go:build !ios

package gamepad

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type nativeGamepadsImpl struct {
	hidManager      _IOHIDManagerRef
	devicesToAdd    []_IOHIDDeviceRef
	devicesToRemove []_IOHIDDeviceRef
	devicesM        sync.Mutex
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (g *nativeGamepadsImpl) init(gamepads *gamepads) error {
	if err := initializeCF(); err != nil {
		return err
	}
	if err := initializeIOKit(); err != nil {
		return err
	}

	var dicts []_CFDictionaryRef

	page := kHIDPage_GenericDesktop
	for _, usage := range []uint{
		kHIDUsage_GD_Joystick,
		kHIDUsage_GD_GamePad,
		kHIDUsage_GD_MultiAxisController,
	} {
		pageRef := _CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, unsafe.Pointer(&page))
		if pageRef == 0 {
			return errors.New("gamepad: CFNumberCreate returned nil")
		}
		defer _CFRelease(_CFTypeRef(pageRef))

		usageRef := _CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, unsafe.Pointer(&usage))
		if usageRef == 0 {
			return errors.New("gamepad: CFNumberCreate returned nil")
		}
		defer _CFRelease(_CFTypeRef(usageRef))

		keys := []_CFStringRef{
			_CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDDeviceUsagePageKey, kCFStringEncodingUTF8),
			_CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDDeviceUsageKey, kCFStringEncodingUTF8),
		}
		values := []_CFNumberRef{
			pageRef,
			usageRef,
		}

		dict := _CFDictionaryCreate(kCFAllocatorDefault,
			(*unsafe.Pointer)(unsafe.Pointer(&keys[0])),
			(*unsafe.Pointer)(unsafe.Pointer(&values[0])),
			_CFIndex(len(keys)), *(**_CFDictionaryKeyCallBacks)(unsafe.Pointer(&kCFTypeDictionaryKeyCallBacks)), *(**_CFDictionaryValueCallBacks)(unsafe.Pointer(&kCFTypeDictionaryValueCallBacks)))
		if dict == 0 {
			return errors.New("gamepad: CFDictionaryCreate returned nil")
		}
		defer _CFRelease(_CFTypeRef(dict))

		dicts = append(dicts, dict)
	}

	matching := _CFArrayCreate(kCFAllocatorDefault,
		(*unsafe.Pointer)(unsafe.Pointer(&dicts[0])),
		_CFIndex(len(dicts)), *(**_CFArrayCallBacks)(unsafe.Pointer(&kCFTypeArrayCallBacks)))
	if matching == 0 {
		return errors.New("gamepad: CFArrayCreateMutable returned nil")
	}
	defer _CFRelease(_CFTypeRef(matching))

	g.hidManager = _IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone)
	if _IOHIDManagerOpen(g.hidManager, kIOHIDOptionsTypeNone) != kIOReturnSuccess {
		return errors.New("gamepad: IOHIDManagerOpen failed")
	}

	_IOHIDManagerSetDeviceMatchingMultiple(g.hidManager, matching)
	_IOHIDManagerRegisterDeviceMatchingCallback(g.hidManager, ebitenGamepadMatchingCallback, nil)
	_IOHIDManagerRegisterDeviceRemovalCallback(g.hidManager, ebitenGamepadRemovalCallback, nil)

	_IOHIDManagerScheduleWithRunLoop(g.hidManager, _CFRunLoopGetMain(), **(**_CFStringRef)(unsafe.Pointer(&kCFRunLoopDefaultMode)))

	// Execute the run loop once in order to register any initially-attached gamepads.
	_CFRunLoopRunInMode(**(**_CFStringRef)(unsafe.Pointer(&kCFRunLoopDefaultMode)), 0, false)

	return nil
}

func ebitenGamepadMatchingCallback(ctx unsafe.Pointer, res _IOReturn, sender unsafe.Pointer, device _IOHIDDeviceRef) {
	n := theGamepads.native.(*nativeGamepadsImpl)
	n.devicesM.Lock()
	defer n.devicesM.Unlock()
	n.devicesToAdd = append(n.devicesToAdd, device)
}

func ebitenGamepadRemovalCallback(ctx unsafe.Pointer, res _IOReturn, sender unsafe.Pointer, device _IOHIDDeviceRef) {
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

func (g *nativeGamepadsImpl) addDevice(device _IOHIDDeviceRef, gamepads *gamepads) {
	if gamepads.find(func(g *Gamepad) bool {
		return g.native.(*nativeGamepadImpl).device == device
	}) != nil {
		return
	}

	elements := _IOHIDDeviceCopyMatchingElements(device, 0, kIOHIDOptionsTypeNone)
	// It is reportedly possible for this to fail on macOS 13 Ventura
	// if the application does not have input monitoring permissions
	if elements == 0 {
		return
	}
	defer _CFRelease(_CFTypeRef(elements))

	name := "Unknown"
	if prop := _IOHIDDeviceGetProperty(device, _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDProductKey, kCFStringEncodingUTF8)); prop != 0 {
		var cstr [256]byte
		_CFStringGetCString(_CFStringRef(prop), cstr[:], kCFStringEncodingUTF8)
		name = strings.TrimRight(string(cstr[:]), "\x00")
	}

	var vendor uint32
	if prop := _IOHIDDeviceGetProperty(device, _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDVendorIDKey, kCFStringEncodingUTF8)); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&vendor))
	}

	var product uint32
	if prop := _IOHIDDeviceGetProperty(device, _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDProductIDKey, kCFStringEncodingUTF8)); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&product))
	}

	var version uint32
	if prop := _IOHIDDeviceGetProperty(device, _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDVersionNumberKey, kCFStringEncodingUTF8)); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&version))
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

	n := &nativeGamepadImpl{
		device: device,
	}
	gp := gamepads.add(name, sdlID)
	gp.native = n

	for i := _CFIndex(0); i < _CFArrayGetCount(elements); i++ {
		native := (_IOHIDElementRef)(_CFArrayGetValueAtIndex(elements, i))
		if _CFGetTypeID(_CFTypeRef(native)) != _IOHIDElementGetTypeID() {
			continue
		}

		typ := _IOHIDElementGetType(native)
		if typ != kIOHIDElementTypeInput_Axis &&
			typ != kIOHIDElementTypeInput_Button &&
			typ != kIOHIDElementTypeInput_Misc {
			continue
		}

		usage := _IOHIDElementGetUsage(native)
		page := _IOHIDElementGetUsagePage(native)

		switch page {
		case kHIDPage_GenericDesktop:
			switch usage {
			case kHIDUsage_GD_X, kHIDUsage_GD_Y, kHIDUsage_GD_Z,
				kHIDUsage_GD_Rx, kHIDUsage_GD_Ry, kHIDUsage_GD_Rz,
				kHIDUsage_GD_Slider, kHIDUsage_GD_Dial, kHIDUsage_GD_Wheel:
				n.axes = append(n.axes, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.axes),
					minimum: int(_IOHIDElementGetLogicalMin(native)),
					maximum: int(_IOHIDElementGetLogicalMax(native)),
				})
			case kHIDUsage_GD_Hatswitch:
				n.hats = append(n.hats, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.hats),
					minimum: int(_IOHIDElementGetLogicalMin(native)),
					maximum: int(_IOHIDElementGetLogicalMax(native)),
				})
			case kHIDUsage_GD_DPadUp, kHIDUsage_GD_DPadRight, kHIDUsage_GD_DPadDown, kHIDUsage_GD_DPadLeft,
				kHIDUsage_GD_SystemMainMenu, kHIDUsage_GD_Select, kHIDUsage_GD_Start:
				n.buttons = append(n.buttons, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.buttons),
					minimum: int(_IOHIDElementGetLogicalMin(native)),
					maximum: int(_IOHIDElementGetLogicalMax(native)),
				})
			}
		case kHIDPage_Simulation:
			switch usage {
			case kHIDUsage_Sim_Accelerator, kHIDUsage_Sim_Brake, kHIDUsage_Sim_Throttle, kHIDUsage_Sim_Rudder, kHIDUsage_Sim_Steering:
				n.axes = append(n.axes, element{
					native:  native,
					usage:   int(usage),
					index:   len(n.axes),
					minimum: int(_IOHIDElementGetLogicalMin(native)),
					maximum: int(_IOHIDElementGetLogicalMax(native)),
				})
			}
		case kHIDPage_Button, kHIDPage_Consumer:
			n.buttons = append(n.buttons, element{
				native:  native,
				usage:   int(usage),
				index:   len(n.buttons),
				minimum: int(_IOHIDElementGetLogicalMin(native)),
				maximum: int(_IOHIDElementGetLogicalMax(native)),
			})
		}
	}

	sort.Stable(n.axes)
	sort.Stable(n.buttons)
	sort.Stable(n.hats)
}

type element struct {
	native  _IOHIDElementRef
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
	device  _IOHIDDeviceRef
	axes    elements
	buttons elements
	hats    elements

	axisValues   []float64
	buttonValues []bool
	hatValues    []int
}

func (g *nativeGamepadImpl) elementValue(e *element) int {
	var valueRef _IOHIDValueRef
	if _IOHIDDeviceGetValue(g.device, e.native, &valueRef) == kIOReturnSuccess {
		return int(_IOHIDValueGetIntegerValue(valueRef))
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

func (*nativeGamepadImpl) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadImpl) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
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

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axisValues) {
		return 0
	}
	return g.axisValues[axis]
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
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

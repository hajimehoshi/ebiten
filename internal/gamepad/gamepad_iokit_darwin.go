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

package gamepad

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type nativeGamepadsIOKit struct {
	hidManager _IOHIDManagerRef

	// devicesToAdd and devicesToRemove hold one reference per entry, which update releases or hands
	// over to the gamepad.
	devicesToAdd    []_IOHIDDeviceRef
	devicesToRemove []_IOHIDDeviceRef
	devicesMu       sync.Mutex

	// deferredDevices maps devices claimed by GameController to their I/O Registry entry IDs
	// (zero if unavailable), holding one reference per device until fallback registration or disconnection.
	deferredDevices map[_IOHIDDeviceRef]uint64
}

// theIOKitGamepads is the running IOKit backend. The C device callbacks reference
// it directly, as theGamepads.native may be a composite backend rather than this one.
var theIOKitGamepads *nativeGamepadsIOKit

func newNativeGamepadsIOKit() nativeGamepads {
	return &nativeGamepadsIOKit{}
}

func (g *nativeGamepadsIOKit) init(gamepads *gamepads) error {
	theIOKitGamepads = g

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

		usagePageKey := _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDDeviceUsagePageKey, kCFStringEncodingUTF8)
		defer _CFRelease(_CFTypeRef(usagePageKey))

		usageKey := _CFStringCreateWithCString(kCFAllocatorDefault, kIOHIDDeviceUsageKey, kCFStringEncodingUTF8)
		defer _CFRelease(_CFTypeRef(usageKey))

		keys := []_CFStringRef{
			usagePageKey,
			usageKey,
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
		_CFRelease(_CFTypeRef(g.hidManager))
		g.hidManager = 0
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

// ebitenGamepadMatchingCallback queues device for update to register. device is only guaranteed to
// stay valid during the callback, so the queued entry takes a reference.
func ebitenGamepadMatchingCallback(ctx unsafe.Pointer, res _IOReturn, sender unsafe.Pointer, device _IOHIDDeviceRef) {
	n := theIOKitGamepads
	n.devicesMu.Lock()
	defer n.devicesMu.Unlock()

	_CFRetain(_CFTypeRef(device))
	n.devicesToAdd = append(n.devicesToAdd, device)
}

// ebitenGamepadRemovalCallback queues device for update to unregister. The queued entry takes a
// reference.
func ebitenGamepadRemovalCallback(ctx unsafe.Pointer, res _IOReturn, sender unsafe.Pointer, device _IOHIDDeviceRef) {
	n := theIOKitGamepads
	n.devicesMu.Lock()
	defer n.devicesMu.Unlock()

	// update only compares the entry with the gamepads by identity, but the reference keeps the
	// address from being reused by a device queued for registration before the removal is applied,
	// which would make the comparison match the new device's gamepad.
	_CFRetain(_CFTypeRef(device))
	n.devicesToRemove = append(n.devicesToRemove, device)
}

func (g *nativeGamepadsIOKit) update(gamepads *gamepads) error {
	g.devicesMu.Lock()
	defer g.devicesMu.Unlock()

	for _, device := range g.devicesToAdd {
		if _, ok := g.deferredDevices[device]; ok {
			_CFRelease(_CFTypeRef(device))
			continue
		}
		if gcSupportsHIDDevice(device) {
			if g.deferredDevices == nil {
				g.deferredDevices = map[_IOHIDDeviceRef]uint64{}
			}
			g.deferredDevices[device] = hidDeviceRegistryID(device)
			continue
		}
		if !g.addDevice(device, gamepads) {
			// No gamepad took over the entry's reference.
			_CFRelease(_CFTypeRef(device))
		}
	}
	for _, device := range g.devicesToRemove {
		if _, ok := g.deferredDevices[device]; ok {
			delete(g.deferredDevices, device)
			_CFRelease(_CFTypeRef(device))
		}
		for {
			gp := gamepads.find(func(gp *Gamepad) bool {
				n, ok := gp.native.(*nativeGamepadHID)
				return ok && n.device == device
			})
			if gp == nil {
				break
			}
			gp.close()
			gamepads.remove(func(gamepad *Gamepad) bool {
				return gamepad == gp
			})
		}
		_CFRelease(_CFTypeRef(device))
	}
	// The GC backend updates first. Its rejection records persist until GC disconnection,
	// so either callback can arrive first, with any number of updates between them.
	for device, id := range g.deferredDevices {
		// A false isKnownRejectedHIDDevice result means no rejected controller has been matched yet:
		// GC may accept the device, its callback or lookup may be pending, or lookup may be unavailable.
		// Keep the device deferred for another check next tick or until disconnection.
		if id == 0 || !theGCGamepads.isKnownRejectedHIDDevice(id) {
			continue
		}
		delete(g.deferredDevices, device)
		if !g.addDevice(device, gamepads) {
			_CFRelease(_CFTypeRef(device))
		}
	}
	g.devicesToAdd = g.devicesToAdd[:0]
	g.devicesToRemove = g.devicesToRemove[:0]
	return nil
}

// hidDeviceProperty returns the device's property for the given null-terminated key name.
// The returned value is owned by the device and must not be released.
func hidDeviceProperty(device _IOHIDDeviceRef, key []byte) _CFTypeRef {
	keyRef := _CFStringCreateWithCString(kCFAllocatorDefault, key, kCFStringEncodingUTF8)
	if keyRef == 0 {
		return 0
	}
	defer _CFRelease(_CFTypeRef(keyRef))
	return _IOHIDDeviceGetProperty(device, keyRef)
}

// addDevice registers a gamepad for device and reports whether it did. The registered gamepad takes
// over the caller's reference to device.
func (g *nativeGamepadsIOKit) addDevice(device _IOHIDDeviceRef, gamepads *gamepads) bool {
	if gamepads.find(func(gp *Gamepad) bool {
		n, ok := gp.native.(*nativeGamepadHID)
		return ok && n.device == device
	}) != nil {
		return false
	}

	elements := _IOHIDDeviceCopyMatchingElements(device, 0, kIOHIDOptionsTypeNone)
	// It is reportedly possible for this to fail on macOS 13 Ventura
	// if the application does not have input monitoring permissions
	if elements == 0 {
		return false
	}
	defer _CFRelease(_CFTypeRef(elements))

	name := "Unknown"
	if prop := hidDeviceProperty(device, kIOHIDProductKey); prop != 0 {
		var cstr [256]byte
		if _CFStringGetCString(_CFStringRef(prop), cstr[:], _CFIndex(len(cstr)), kCFStringEncodingUTF8) {
			name = strings.TrimRight(string(cstr[:]), "\x00")
		}
	}

	var vendor uint32
	if prop := hidDeviceProperty(device, kIOHIDVendorIDKey); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&vendor))
	}

	var product uint32
	if prop := hidDeviceProperty(device, kIOHIDProductIDKey); prop != 0 {
		_CFNumberGetValue(_CFNumberRef(prop), kCFNumberSInt32Type, unsafe.Pointer(&product))
	}

	var version uint32
	if prop := hidDeviceProperty(device, kIOHIDVersionNumberKey); prop != 0 {
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

	n := &nativeGamepadHID{
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

	slices.SortStableFunc(n.axes, compareElements)
	slices.SortStableFunc(n.buttons, compareElements)
	slices.SortStableFunc(n.hats, compareElements)
	return true
}

func compareElements(a, b element) int {
	return cmp.Or(
		cmp.Compare(a.usage, b.usage),
		cmp.Compare(a.index, b.index),
	)
}

type element struct {
	native  _IOHIDElementRef
	usage   int
	index   int
	minimum int
	maximum int
}

type nativeGamepadHID struct {
	device  _IOHIDDeviceRef
	axes    []element
	buttons []element
	hats    []element

	axisValues   []float64
	buttonValues []bool
	hatValues    []int
}

// close releases g's native resources. close can be called multiple times.
func (g *nativeGamepadHID) close() {
	if g.device == 0 {
		return
	}
	_CFRelease(_CFTypeRef(g.device))
	g.device = 0
}

func (g *nativeGamepadHID) elementValue(e *element) int {
	var valueRef _IOHIDValueRef
	if _IOHIDDeviceGetValue(g.device, e.native, &valueRef) == kIOReturnSuccess {
		return int(_IOHIDValueGetIntegerValue(valueRef))
	}
	return 0
}

func (g *nativeGamepadHID) update(gamepads *gamepads) error {
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

	for i := range g.axes {
		a := &g.axes[i]
		raw := g.elementValue(a)
		a.minimum = min(a.minimum, raw)
		a.maximum = max(a.maximum, raw)
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

func (g *nativeGamepadHID) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadHID) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadHID) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadHID) axisCount() int {
	return len(g.axisValues)
}

func (g *nativeGamepadHID) buttonCount() int {
	return len(g.buttonValues)
}

func (g *nativeGamepadHID) hatCount() int {
	return len(g.hatValues)
}

func (g *nativeGamepadHID) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadHID) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axisValues) {
		return 0
	}
	return g.axisValues[axis]
}

func (g *nativeGamepadHID) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadHID) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttonValues) {
		return false
	}
	return g.buttonValues[button]
}

func (g *nativeGamepadHID) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hatValues) {
		return hatCentered
	}
	return g.hatValues[hat]
}

func (g *nativeGamepadHID) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}

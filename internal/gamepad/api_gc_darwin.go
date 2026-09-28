// Copyright 2021 The Ebiten Authors
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
	"github.com/ebitengine/purego/objc"
)

// ObjC classes (initialized in init after loading GameController framework).
var (
	class_GCController         objc.Class
	class_NSNotificationCenter objc.Class
)

// ObjC selectors for GameController framework.
var (
	sel_controllers                                objc.SEL
	sel_extendedGamepad                            objc.SEL
	sel_productCategory                            objc.SEL
	sel_vendorName                                 objc.SEL
	sel_physicalInputProfile                       objc.SEL
	sel_respondsToSelector                         objc.SEL
	sel_isEqualToString                            objc.SEL
	sel_leftThumbstick                             objc.SEL
	sel_rightThumbstick                            objc.SEL
	sel_leftThumbstickButton                       objc.SEL
	sel_rightThumbstickButton                      objc.SEL
	sel_buttonA                                    objc.SEL
	sel_buttonB                                    objc.SEL
	sel_buttonX                                    objc.SEL
	sel_buttonY                                    objc.SEL
	sel_leftShoulder                               objc.SEL
	sel_rightShoulder                              objc.SEL
	sel_buttonOptions                              objc.SEL
	sel_buttonHome                                 objc.SEL
	sel_buttonMenu                                 objc.SEL
	sel_leftTrigger                                objc.SEL
	sel_rightTrigger                               objc.SEL
	sel_dpad                                       objc.SEL
	sel_xAxis                                      objc.SEL
	sel_yAxis                                      objc.SEL
	sel_value                                      objc.SEL
	sel_isPressed                                  objc.SEL
	sel_up                                         objc.SEL
	sel_down                                       objc.SEL
	sel_left                                       objc.SEL
	sel_right                                      objc.SEL
	sel_buttons                                    objc.SEL
	sel_objectForKeyedSubscript                    objc.SEL
	sel_object                                     objc.SEL
	sel_defaultCenter                              objc.SEL
	sel_addObserverForName_object_queue_usingBlock objc.SEL
	sel_alloc                                      objc.SEL
	sel_initWithUTF8String                         objc.SEL
	sel_count                                      objc.SEL
	sel_objectAtIndex                              objc.SEL
	sel_supportsHIDDevice                          objc.SEL
	sel_dpads                                      objc.SEL
	sel_touchpads                                  objc.SEL
	sel_touchSurface                               objc.SEL
	sel_touchState                                 objc.SEL
	sel_setValueChangedHandler                     objc.SEL
	sel_setTouchDown                               objc.SEL
	sel_setTouchMoved                              objc.SEL
	sel_setTouchUp                                 objc.SEL
	sel_retain                                     objc.SEL
	sel_release                                    objc.SEL
)

// GC notification and input string constants (loaded from framework symbols).
var (
	gcControllerDidConnectNotification    uintptr
	gcControllerDidDisconnectNotification uintptr
	gcInputDualShockTouchpadButton        objc.ID
	gcInputXboxPaddleOne                  objc.ID
	gcInputXboxPaddleTwo                  objc.ID
	gcInputXboxPaddleThree                objc.ID
	gcInputXboxPaddleFour                 objc.ID
	gcInputXboxShareButton                objc.ID // "Button Share"
	gcInputDualShockTouchpadOne           objc.ID
	gcInputDualShockTouchpadTwo           objc.ID
)

func init() {
	// Load GameController framework.
	gc, err := purego.Dlopen("/System/Library/Frameworks/GameController.framework/GameController", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		// GameController might not be available; skip initialization.
		return
	}

	class_GCController = objc.GetClass("GCController")
	class_NSNotificationCenter = objc.GetClass("NSNotificationCenter")

	sel_controllers = objc.RegisterName("controllers")
	sel_extendedGamepad = objc.RegisterName("extendedGamepad")
	sel_productCategory = objc.RegisterName("productCategory")
	sel_vendorName = objc.RegisterName("vendorName")
	sel_physicalInputProfile = objc.RegisterName("physicalInputProfile")
	sel_respondsToSelector = objc.RegisterName("respondsToSelector:")
	sel_isEqualToString = objc.RegisterName("isEqualToString:")
	sel_leftThumbstick = objc.RegisterName("leftThumbstick")
	sel_rightThumbstick = objc.RegisterName("rightThumbstick")
	sel_leftThumbstickButton = objc.RegisterName("leftThumbstickButton")
	sel_rightThumbstickButton = objc.RegisterName("rightThumbstickButton")
	sel_buttonA = objc.RegisterName("buttonA")
	sel_buttonB = objc.RegisterName("buttonB")
	sel_buttonX = objc.RegisterName("buttonX")
	sel_buttonY = objc.RegisterName("buttonY")
	sel_leftShoulder = objc.RegisterName("leftShoulder")
	sel_rightShoulder = objc.RegisterName("rightShoulder")
	sel_buttonOptions = objc.RegisterName("buttonOptions")
	sel_buttonHome = objc.RegisterName("buttonHome")
	sel_buttonMenu = objc.RegisterName("buttonMenu")
	sel_leftTrigger = objc.RegisterName("leftTrigger")
	sel_rightTrigger = objc.RegisterName("rightTrigger")
	sel_dpad = objc.RegisterName("dpad")
	sel_xAxis = objc.RegisterName("xAxis")
	sel_yAxis = objc.RegisterName("yAxis")
	sel_value = objc.RegisterName("value")
	sel_isPressed = objc.RegisterName("isPressed")
	sel_up = objc.RegisterName("up")
	sel_down = objc.RegisterName("down")
	sel_left = objc.RegisterName("left")
	sel_right = objc.RegisterName("right")
	sel_buttons = objc.RegisterName("buttons")
	sel_objectForKeyedSubscript = objc.RegisterName("objectForKeyedSubscript:")
	sel_object = objc.RegisterName("object")
	sel_defaultCenter = objc.RegisterName("defaultCenter")
	sel_addObserverForName_object_queue_usingBlock = objc.RegisterName("addObserverForName:object:queue:usingBlock:")
	sel_alloc = objc.RegisterName("alloc")
	sel_initWithUTF8String = objc.RegisterName("initWithUTF8String:")
	sel_count = objc.RegisterName("count")
	sel_objectAtIndex = objc.RegisterName("objectAtIndex:")
	sel_supportsHIDDevice = objc.RegisterName("supportsHIDDevice:")
	sel_dpads = objc.RegisterName("dpads")
	sel_touchpads = objc.RegisterName("touchpads")
	sel_touchSurface = objc.RegisterName("touchSurface")
	sel_touchState = objc.RegisterName("touchState")
	sel_setValueChangedHandler = objc.RegisterName("setValueChangedHandler:")
	sel_setTouchDown = objc.RegisterName("setTouchDown:")
	sel_setTouchMoved = objc.RegisterName("setTouchMoved:")
	sel_setTouchUp = objc.RegisterName("setTouchUp:")
	sel_retain = objc.RegisterName("retain")
	sel_release = objc.RegisterName("release")

	// Load notification name symbols (NSString* globals).
	connectPtr, err := purego.Dlsym(gc, "GCControllerDidConnectNotification")
	if err == nil {
		gcControllerDidConnectNotification = connectPtr
	}
	disconnectPtr, err := purego.Dlsym(gc, "GCControllerDidDisconnectNotification")
	if err == nil {
		gcControllerDidDisconnectNotification = disconnectPtr
	}

	// Load GCInput string constants (NSString* globals, available macOS 10.15+).
	loadNSStringSymbol := func(name string) objc.ID {
		ptr, err := purego.Dlsym(gc, name)
		if err != nil {
			return 0
		}
		// The symbol is a pointer to an NSString*.
		return *(*objc.ID)(unsafe.Pointer(ptr))
	}

	gcInputDualShockTouchpadButton = loadNSStringSymbol("GCInputDualShockTouchpadButton")
	gcInputXboxPaddleOne = loadNSStringSymbol("GCInputXboxPaddleOne")
	gcInputXboxPaddleTwo = loadNSStringSymbol("GCInputXboxPaddleTwo")
	gcInputXboxPaddleThree = loadNSStringSymbol("GCInputXboxPaddleThree")
	gcInputXboxPaddleFour = loadNSStringSymbol("GCInputXboxPaddleFour")
	gcInputDualShockTouchpadOne = loadNSStringSymbol("GCInputDualShockTouchpadOne")
	gcInputDualShockTouchpadTwo = loadNSStringSymbol("GCInputDualShockTouchpadTwo")

	// GCInputXboxShareButton is not an official constant; use "Button Share".
	classNSString := objc.GetClass("NSString")
	gcInputXboxShareButton = objc.ID(classNSString).Send(sel_alloc).Send(sel_initWithUTF8String, "Button Share\x00")
}

// nsStringEquals checks if an NSString equals a Go string.
func nsStringEquals(nsStr objc.ID, s string) bool {
	if nsStr == 0 {
		return false
	}
	classNSString := objc.GetClass("NSString")
	goNSStr := objc.ID(classNSString).Send(sel_alloc).Send(sel_initWithUTF8String, s+"\x00")
	defer goNSStr.Send(objc.RegisterName("release"))
	return nsStr.Send(sel_isEqualToString, goNSStr) != 0
}

// getAxisValue reads a float value from an ObjC axis element (returns the raw float32 value from the `value` property).
func getAxisValue(element objc.ID) float32 {
	return objc.Send[float32](element, sel_value)
}

// getIsPressed reads the boolean isPressed property from an ObjC button element.
func getIsPressed(element objc.ID) bool {
	return element.Send(sel_isPressed) != 0
}

// gcSupportsHIDDevice reports whether the GameController framework claims the given HID device.
// It reports false when GameController or +[GCController supportsHIDDevice:] (macOS 11+) is unavailable.
func gcSupportsHIDDevice(device _IOHIDDeviceRef) bool {
	if class_GCController == 0 {
		return false
	}
	gcClass := objc.ID(class_GCController)
	if gcClass.Send(sel_respondsToSelector, sel_supportsHIDDevice) == 0 {
		return false
	}
	return gcClass.Send(sel_supportsHIDDevice, uintptr(device)) != 0
}

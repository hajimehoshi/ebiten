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
	"encoding/hex"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Controller button constants.
const (
	kControllerButtonA = iota
	kControllerButtonB
	kControllerButtonX
	kControllerButtonY
	kControllerButtonBack
	kControllerButtonGuide
	kControllerButtonStart
	kControllerButtonLeftStick
	kControllerButtonRightStick
	kControllerButtonLeftShoulder
	kControllerButtonRightShoulder
	kControllerButtonDpadUp
	kControllerButtonDpadDown
	kControllerButtonDpadLeft
	kControllerButtonDpadRight
	kControllerButtonMisc1
	kControllerButtonPaddle1
	kControllerButtonPaddle2
	kControllerButtonPaddle3
	kControllerButtonPaddle4
	kControllerButtonTouchpad
	kControllerButtonMax
)

// Hat state constants.
const (
	kHatCentered  uint8 = 0x00
	kHatUp        uint8 = 0x01
	kHatRight     uint8 = 0x02
	kHatDown      uint8 = 0x04
	kHatLeft      uint8 = 0x08
	kHatRightUp         = kHatRight | kHatUp
	kHatRightDown       = kHatRight | kHatDown
	kHatLeftUp          = kHatLeft | kHatUp
	kHatLeftDown        = kHatLeft | kHatDown
)

// USB vendor IDs.
const (
	kUSBVendorApple     uint16 = 0x05ac
	kUSBVendorMicrosoft uint16 = 0x045e
	kUSBVendorSony      uint16 = 0x054c
)

// USB product IDs.
const (
	kUSBProductSonyDS4Slim                  uint16 = 0x09cc
	kUSBProductSonyDS5                      uint16 = 0x0ce6
	kUSBProductXboxOneEliteSeries2Bluetooth uint16 = 0x0b05
	kUSBProductXboxOneSRev1Bluetooth        uint16 = 0x02e0
	kUSBProductXboxSeriesXBluetooth         uint16 = 0x0b13
)

// SDL hardware bus type.
const kSDLHardwareBusBluetooth uint16 = 0x05

// controllerProperty holds extracted controller metadata.
type controllerProperty struct {
	nAxes                uint8
	nButtons             uint8
	nHats                uint8
	buttonMask           uint32
	guid                 [16]byte
	name                 string
	hasDualshockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool
}

// controllerState holds the current input state of a controller.
type controllerState struct {
	buttons [32]uint8
	axes    [32]float32
	hat     uint8
}

// ObjC classes (initialized in init after loading GameController framework).
var (
	class_GCController         objc.Class
	class_NSNotificationCenter objc.Class
)

// ObjC selectors for GameController framework.
var (
	sel_controllers                                objc.SEL
	sel_extendedGamepad                            objc.SEL
	sel_microGamepad                               objc.SEL
	sel_productCategory                            objc.SEL
	sel_vendorName                                 objc.SEL
	sel_physicalInputProfile                       objc.SEL
	sel_respondsToSelector                         objc.SEL
	sel_isEqualToString                            objc.SEL
	sel_UTF8String                                 objc.SEL
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
	sel_microGamepad = objc.RegisterName("microGamepad")
	sel_productCategory = objc.RegisterName("productCategory")
	sel_vendorName = objc.RegisterName("vendorName")
	sel_physicalInputProfile = objc.RegisterName("physicalInputProfile")
	sel_respondsToSelector = objc.RegisterName("respondsToSelector:")
	sel_isEqualToString = objc.RegisterName("isEqualToString:")
	sel_UTF8String = objc.RegisterName("UTF8String")
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

	// GCInputXboxShareButton is not an official constant; use "Button Share".
	classNSString := objc.GetClass("NSString")
	gcInputXboxShareButton = objc.ID(classNSString).Send(sel_alloc).Send(sel_initWithUTF8String, "Button Share\x00")
}

// nsStringToGoString converts an ObjC NSString to a Go string.
func nsStringToGoString(nsStr objc.ID) string {
	if nsStr == 0 {
		return ""
	}
	ptr := nsStr.Send(sel_UTF8String)
	if ptr == 0 {
		return ""
	}
	// Read a C string from the pointer.
	cstr := unsafe.Pointer(ptr)
	length := 0
	for *(*byte)(unsafe.Add(cstr, length)) != 0 {
		length++
	}
	return string(unsafe.Slice((*byte)(cstr), length))
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

// getControllerPropertyFromController extracts controller properties via ObjC.
func getControllerPropertyFromController(controller objc.ID) controllerProperty {
	var prop controllerProperty

	// Get controller name.
	vendorNameStr := controller.Send(sel_vendorName)
	if vendorNameStr != 0 {
		prop.name = nsStringToGoString(vendorNameStr)
	}
	if prop.name == "" {
		prop.name = "MFi Gamepad"
	}

	var vendor, product, subtype uint16

	extGamepad := controller.Send(sel_extendedGamepad)
	if extGamepad != 0 {
		// Detect controller type via productCategory (macOS 10.15+) or vendorName.
		isXbox := false
		isPS4 := false
		isPS5 := false

		productCategory := controller.Send(sel_productCategory)
		if productCategory != 0 {
			if nsStringEquals(productCategory, "DualShock 4") {
				isPS4 = true
			} else if nsStringEquals(productCategory, "DualSense") {
				isPS5 = true
			} else if nsStringEquals(productCategory, "Xbox One") {
				isXbox = true
			}
		}
		if !isXbox && !isPS4 && !isPS5 {
			vendorName := controller.Send(sel_vendorName)
			if vendorName != 0 {
				if nsStringEquals(vendorName, "DUALSHOCK") {
					isPS4 = true
				} else if nsStringEquals(vendorName, "DualSense") {
					isPS5 = true
				} else if nsStringEquals(vendorName, "Xbox") {
					isXbox = true
				}
			}
		}

		// Standard buttons.
		prop.buttonMask |= (1 << kControllerButtonA)
		prop.buttonMask |= (1 << kControllerButtonB)
		prop.buttonMask |= (1 << kControllerButtonX)
		prop.buttonMask |= (1 << kControllerButtonY)
		prop.buttonMask |= (1 << kControllerButtonLeftShoulder)
		prop.buttonMask |= (1 << kControllerButtonRightShoulder)
		prop.nButtons += 6

		// Optional buttons (check availability via respondsToSelector:).
		if extGamepad.Send(sel_respondsToSelector, sel_leftThumbstickButton) != 0 && extGamepad.Send(sel_leftThumbstickButton) != 0 {
			prop.buttonMask |= (1 << kControllerButtonLeftStick)
			prop.nButtons++
		}
		if extGamepad.Send(sel_respondsToSelector, sel_rightThumbstickButton) != 0 && extGamepad.Send(sel_rightThumbstickButton) != 0 {
			prop.buttonMask |= (1 << kControllerButtonRightStick)
			prop.nButtons++
		}
		if extGamepad.Send(sel_respondsToSelector, sel_buttonOptions) != 0 && extGamepad.Send(sel_buttonOptions) != 0 {
			prop.buttonMask |= (1 << kControllerButtonBack)
			prop.nButtons++
		}
		if extGamepad.Send(sel_respondsToSelector, sel_buttonHome) != 0 && extGamepad.Send(sel_buttonHome) != 0 {
			prop.buttonMask |= (1 << kControllerButtonGuide)
			prop.nButtons++
		}

		prop.buttonMask |= (1 << kControllerButtonStart)
		prop.nButtons++

		// Physical input profile buttons (GCInputDualShockTouchpad, Xbox paddles, etc.).
		if controller.Send(sel_respondsToSelector, sel_physicalInputProfile) != 0 {
			profile := controller.Send(sel_physicalInputProfile)
			if profile != 0 {
				profileButtons := profile.Send(sel_buttons)
				if profileButtons != 0 {
					if gcInputDualShockTouchpadButton != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputDualShockTouchpadButton) != 0 {
						prop.hasDualshockTouchpad = true
						prop.buttonMask |= (1 << kControllerButtonMisc1)
						prop.nButtons++
					}
					if gcInputXboxPaddleOne != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleOne) != 0 {
						prop.hasXboxPaddles = true
						prop.buttonMask |= (1 << kControllerButtonPaddle1)
						prop.nButtons++
					}
					if gcInputXboxPaddleTwo != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleTwo) != 0 {
						prop.hasXboxPaddles = true
						prop.buttonMask |= (1 << kControllerButtonPaddle2)
						prop.nButtons++
					}
					if gcInputXboxPaddleThree != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleThree) != 0 {
						prop.hasXboxPaddles = true
						prop.buttonMask |= (1 << kControllerButtonPaddle3)
						prop.nButtons++
					}
					if gcInputXboxPaddleFour != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleFour) != 0 {
						prop.hasXboxPaddles = true
						prop.buttonMask |= (1 << kControllerButtonPaddle4)
						prop.nButtons++
					}
					if gcInputXboxShareButton != 0 && profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxShareButton) != 0 {
						prop.hasXboxShareButton = true
						prop.buttonMask |= (1 << kControllerButtonMisc1)
						prop.nButtons++
					}
				}
			}
		}

		// Determine vendor/product/subtype for GUID.
		if isXbox {
			vendor = kUSBVendorMicrosoft
			if prop.hasXboxPaddles {
				product = kUSBProductXboxOneEliteSeries2Bluetooth
				subtype = 1
			} else if prop.hasXboxShareButton {
				product = kUSBProductXboxSeriesXBluetooth
				subtype = 1
			} else {
				product = kUSBProductXboxOneSRev1Bluetooth
				subtype = 0
			}
		} else if isPS4 {
			vendor = kUSBVendorSony
			product = kUSBProductSonyDS4Slim
			if prop.hasDualshockTouchpad {
				subtype = 1
			}
		} else if isPS5 {
			vendor = kUSBVendorSony
			product = kUSBProductSonyDS5
			subtype = 0
		} else {
			vendor = kUSBVendorApple
			product = 1
			subtype = 1
		}

		prop.nAxes = 6
		prop.nHats = 1
	}

	// Build GUID (SDL-compatible format).
	prop.guid[0] = byte(kSDLHardwareBusBluetooth)
	prop.guid[1] = byte(kSDLHardwareBusBluetooth >> 8)
	prop.guid[4] = byte(vendor)
	prop.guid[5] = byte(vendor >> 8)
	prop.guid[8] = byte(product)
	prop.guid[9] = byte(product >> 8)
	prop.guid[12] = byte(prop.buttonMask)
	prop.guid[13] = byte(prop.buttonMask >> 8)
	if vendor == kUSBVendorApple {
		prop.guid[14] = 'm'
	}
	prop.guid[15] = byte(subtype)

	return prop
}

// getAxisValue reads a float value from an ObjC axis element (returns the raw float32 value from the `value` property).
func getAxisValue(element objc.ID) float32 {
	return objc.Send[float32](element, sel_value)
}

// getIsPressed reads the boolean isPressed property from an ObjC button element.
func getIsPressed(element objc.ID) bool {
	return element.Send(sel_isPressed) != 0
}

// getHatState reads the hat state from a GCControllerDirectionPad.
func getHatState(dpad objc.ID) uint8 {
	var hat uint8
	if getIsPressed(dpad.Send(sel_up)) {
		hat |= kHatUp
	} else if getIsPressed(dpad.Send(sel_down)) {
		hat |= kHatDown
	}
	if getIsPressed(dpad.Send(sel_left)) {
		hat |= kHatLeft
	} else if getIsPressed(dpad.Send(sel_right)) {
		hat |= kHatRight
	}
	return hat
}

// getControllerStateGC reads the current input state from a GCController.
func getControllerStateGC(controllerPtr uintptr, buttonMask uint32, nHats int,
	hasDualshockTouchpad, hasXboxPaddles, hasXboxShareButton bool) controllerState {

	controller := objc.ID(controllerPtr)
	var state controllerState

	extGamepad := controller.Send(sel_extendedGamepad)
	if extGamepad == 0 {
		return state
	}

	// Axes.
	leftStick := extGamepad.Send(sel_leftThumbstick)
	rightStick := extGamepad.Send(sel_rightThumbstick)
	state.axes[0] = getAxisValue(leftStick.Send(sel_xAxis))
	state.axes[1] = -getAxisValue(leftStick.Send(sel_yAxis))
	state.axes[2] = getAxisValue(extGamepad.Send(sel_leftTrigger))*2 - 1
	state.axes[3] = getAxisValue(rightStick.Send(sel_xAxis))
	state.axes[4] = -getAxisValue(rightStick.Send(sel_yAxis))
	state.axes[5] = getAxisValue(extGamepad.Send(sel_rightTrigger))*2 - 1

	// Buttons.
	buttonCount := 0
	setButton := func(pressed bool) {
		if pressed {
			state.buttons[buttonCount] = 1
		}
		buttonCount++
	}
	setButton(getIsPressed(extGamepad.Send(sel_buttonA)))
	setButton(getIsPressed(extGamepad.Send(sel_buttonB)))
	setButton(getIsPressed(extGamepad.Send(sel_buttonX)))
	setButton(getIsPressed(extGamepad.Send(sel_buttonY)))
	setButton(getIsPressed(extGamepad.Send(sel_leftShoulder)))
	setButton(getIsPressed(extGamepad.Send(sel_rightShoulder)))

	if buttonMask&(1<<kControllerButtonLeftStick) != 0 {
		setButton(getIsPressed(extGamepad.Send(sel_leftThumbstickButton)))
	}
	if buttonMask&(1<<kControllerButtonRightStick) != 0 {
		setButton(getIsPressed(extGamepad.Send(sel_rightThumbstickButton)))
	}
	if buttonMask&(1<<kControllerButtonBack) != 0 {
		setButton(getIsPressed(extGamepad.Send(sel_buttonOptions)))
	}
	if buttonMask&(1<<kControllerButtonGuide) != 0 {
		setButton(getIsPressed(extGamepad.Send(sel_buttonHome)))
	}
	if buttonMask&(1<<kControllerButtonStart) != 0 {
		setButton(getIsPressed(extGamepad.Send(sel_buttonMenu)))
	}

	if hasDualshockTouchpad {
		profile := controller.Send(sel_physicalInputProfile)
		profileButtons := profile.Send(sel_buttons)
		btn := profileButtons.Send(sel_objectForKeyedSubscript, gcInputDualShockTouchpadButton)
		setButton(getIsPressed(btn))
	}
	if hasXboxPaddles {
		profile := controller.Send(sel_physicalInputProfile)
		profileButtons := profile.Send(sel_buttons)
		if buttonMask&(1<<kControllerButtonPaddle1) != 0 {
			setButton(getIsPressed(profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleOne)))
		}
		if buttonMask&(1<<kControllerButtonPaddle2) != 0 {
			setButton(getIsPressed(profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleTwo)))
		}
		if buttonMask&(1<<kControllerButtonPaddle3) != 0 {
			setButton(getIsPressed(profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleThree)))
		}
		if buttonMask&(1<<kControllerButtonPaddle4) != 0 {
			setButton(getIsPressed(profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxPaddleFour)))
		}
	}
	if hasXboxShareButton {
		profile := controller.Send(sel_physicalInputProfile)
		profileButtons := profile.Send(sel_buttons)
		setButton(getIsPressed(profileButtons.Send(sel_objectForKeyedSubscript, gcInputXboxShareButton)))
	}

	// Hat.
	if nHats > 0 {
		state.hat = getHatState(extGamepad.Send(sel_dpad))
	}

	return state
}

// addController adds a GCController to the gamepad list.
func addController(controller objc.ID) {
	// Ignore if the controller is not an actual controller (e.g., Siri Remote).
	if controller.Send(sel_extendedGamepad) == 0 && controller.Send(sel_microGamepad) != 0 {
		return
	}

	prop := getControllerPropertyFromController(controller)
	theGamepads.addGCGamepad(uintptr(controller), prop)
}

// removeController removes a GCController from the gamepad list.
func removeController(controller objc.ID) {
	theGamepads.removeGCGamepad(uintptr(controller))
}

func (g *gamepads) addGCGamepad(controller uintptr, prop controllerProperty) {
	g.m.Lock()
	defer g.m.Unlock()

	sdlID := hex.EncodeToString(prop.guid[:])
	gp := g.add(prop.name, sdlID)
	gp.native = &nativeGamepadGC{
		controller:           controller,
		axes:                 make([]float64, prop.nAxes),
		buttons:              make([]bool, prop.nButtons+prop.nHats*4),
		hats:                 make([]int, prop.nHats),
		buttonMask:           prop.buttonMask,
		hasDualshockTouchpad: prop.hasDualshockTouchpad,
		hasXboxPaddles:       prop.hasXboxPaddles,
		hasXboxShareButton:   prop.hasXboxShareButton,
		leftMotor:            createGCRumbleMotor(controller, 0),
		rightMotor:           createGCRumbleMotor(controller, 1),
	}
}

func (g *gamepads) removeGCGamepad(controller uintptr) {
	g.m.Lock()
	defer g.m.Unlock()

	g.remove(func(gamepad *Gamepad) bool {
		gc := gamepad.native.(*nativeGamepadGC)
		if gc.controller == controller {
			releaseGCRumbleMotor(gc.leftMotor)
			releaseGCRumbleMotor(gc.rightMotor)
			return true
		}
		return false
	})
}

func initializeGCGamepads() {
	if class_GCController == 0 {
		return
	}

	// Add all currently connected controllers.
	controllers := objc.ID(class_GCController).Send(sel_controllers)
	count := int(controllers.Send(sel_count))
	for i := range count {
		controller := controllers.Send(sel_objectAtIndex, i)
		addController(controller)
	}

	// Register for connect/disconnect notifications.
	center := objc.ID(class_NSNotificationCenter).Send(sel_defaultCenter)

	connectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
		controller := notification.Send(sel_object)
		addController(controller)
	})

	disconnectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
		controller := notification.Send(sel_object)
		removeController(controller)
	})

	// The notification name symbols are pointers to NSString* — dereference them.
	if gcControllerDidConnectNotification != 0 {
		connectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidConnectNotification))
		center.Send(sel_addObserverForName_object_queue_usingBlock, connectName, uintptr(0), uintptr(0), connectBlock)
	}
	if gcControllerDidDisconnectNotification != 0 {
		disconnectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidDisconnectNotification))
		center.Send(sel_addObserverForName_object_queue_usingBlock, disconnectName, uintptr(0), uintptr(0), disconnectBlock)
	}
}

func (g *nativeGamepadGC) updateGCGamepad() {
	state := getControllerStateGC(g.controller, g.buttonMask, len(g.hats),
		g.hasDualshockTouchpad, g.hasXboxPaddles, g.hasXboxShareButton)

	nButtons := len(g.buttons) - len(g.hats)*4
	for i := range nButtons {
		g.buttons[i] = state.buttons[i] != 0
	}

	// Follow the GLFW way to process hats.
	if len(g.hats) > 0 {
		base := len(g.buttons) - len(g.hats)*4
		g.buttons[base] = state.hat&0x01 != 0
		g.buttons[base+1] = state.hat&0x02 != 0
		g.buttons[base+2] = state.hat&0x04 != 0
		g.buttons[base+3] = state.hat&0x08 != 0
	}

	for i := range g.axes {
		g.axes[i] = float64(state.axes[i])
	}

	if len(g.hats) > 0 {
		g.hats[0] = int(state.hat)
	}
}

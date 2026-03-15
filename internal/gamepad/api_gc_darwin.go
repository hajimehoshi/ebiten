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
	classGCController      objc.Class
	classNSNotificationCtr objc.Class
)

// ObjC selectors for GameController framework.
var (
	selControllers             objc.SEL
	selExtendedGamepad         objc.SEL
	selMicroGamepad            objc.SEL
	selProductCategory         objc.SEL
	selVendorName              objc.SEL
	selPhysicalInputProfile    objc.SEL
	selRespondsToSelector      objc.SEL
	selIsEqualToString         objc.SEL
	selUTF8String              objc.SEL
	selLeftThumbstick          objc.SEL
	selRightThumbstick         objc.SEL
	selLeftThumbstickButton    objc.SEL
	selRightThumbstickButton   objc.SEL
	selButtonA                 objc.SEL
	selButtonB                 objc.SEL
	selButtonX                 objc.SEL
	selButtonY                 objc.SEL
	selLeftShoulder            objc.SEL
	selRightShoulder           objc.SEL
	selButtonOptions           objc.SEL
	selButtonHome              objc.SEL
	selButtonMenu              objc.SEL
	selLeftTrigger             objc.SEL
	selRightTrigger            objc.SEL
	selDpad                    objc.SEL
	selXAxis                   objc.SEL
	selYAxis                   objc.SEL
	selValue                   objc.SEL
	selIsPressed               objc.SEL
	selUp                      objc.SEL
	selDown                    objc.SEL
	selLeft                    objc.SEL
	selRight                   objc.SEL
	selButtons                 objc.SEL
	selObjectForKeyedSubscript objc.SEL
	selObject                  objc.SEL
	selDefaultCenter           objc.SEL
	selAddObserver             objc.SEL
	selAlloc                   objc.SEL
	selInitWithUTF8String      objc.SEL
	selCount                   objc.SEL
	selObjectAtIndex           objc.SEL
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

	classGCController = objc.GetClass("GCController")
	classNSNotificationCtr = objc.GetClass("NSNotificationCenter")

	selControllers = objc.RegisterName("controllers")
	selExtendedGamepad = objc.RegisterName("extendedGamepad")
	selMicroGamepad = objc.RegisterName("microGamepad")
	selProductCategory = objc.RegisterName("productCategory")
	selVendorName = objc.RegisterName("vendorName")
	selPhysicalInputProfile = objc.RegisterName("physicalInputProfile")
	selRespondsToSelector = objc.RegisterName("respondsToSelector:")
	selIsEqualToString = objc.RegisterName("isEqualToString:")
	selUTF8String = objc.RegisterName("UTF8String")
	selLeftThumbstick = objc.RegisterName("leftThumbstick")
	selRightThumbstick = objc.RegisterName("rightThumbstick")
	selLeftThumbstickButton = objc.RegisterName("leftThumbstickButton")
	selRightThumbstickButton = objc.RegisterName("rightThumbstickButton")
	selButtonA = objc.RegisterName("buttonA")
	selButtonB = objc.RegisterName("buttonB")
	selButtonX = objc.RegisterName("buttonX")
	selButtonY = objc.RegisterName("buttonY")
	selLeftShoulder = objc.RegisterName("leftShoulder")
	selRightShoulder = objc.RegisterName("rightShoulder")
	selButtonOptions = objc.RegisterName("buttonOptions")
	selButtonHome = objc.RegisterName("buttonHome")
	selButtonMenu = objc.RegisterName("buttonMenu")
	selLeftTrigger = objc.RegisterName("leftTrigger")
	selRightTrigger = objc.RegisterName("rightTrigger")
	selDpad = objc.RegisterName("dpad")
	selXAxis = objc.RegisterName("xAxis")
	selYAxis = objc.RegisterName("yAxis")
	selValue = objc.RegisterName("value")
	selIsPressed = objc.RegisterName("isPressed")
	selUp = objc.RegisterName("up")
	selDown = objc.RegisterName("down")
	selLeft = objc.RegisterName("left")
	selRight = objc.RegisterName("right")
	selButtons = objc.RegisterName("buttons")
	selObjectForKeyedSubscript = objc.RegisterName("objectForKeyedSubscript:")
	selObject = objc.RegisterName("object")
	selDefaultCenter = objc.RegisterName("defaultCenter")
	selAddObserver = objc.RegisterName("addObserverForName:object:queue:usingBlock:")
	selAlloc = objc.RegisterName("alloc")
	selInitWithUTF8String = objc.RegisterName("initWithUTF8String:")
	selCount = objc.RegisterName("count")
	selObjectAtIndex = objc.RegisterName("objectAtIndex:")

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
	gcInputXboxShareButton = objc.ID(classNSString).Send(selAlloc).Send(selInitWithUTF8String, "Button Share\x00")
}

// nsStringToGoString converts an ObjC NSString to a Go string.
func nsStringToGoString(nsStr objc.ID) string {
	if nsStr == 0 {
		return ""
	}
	ptr := nsStr.Send(selUTF8String)
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
	goNSStr := objc.ID(classNSString).Send(selAlloc).Send(selInitWithUTF8String, s+"\x00")
	defer goNSStr.Send(objc.RegisterName("release"))
	return nsStr.Send(selIsEqualToString, goNSStr) != 0
}

// getControllerPropertyFromController extracts controller properties via ObjC.
func getControllerPropertyFromController(controller objc.ID) controllerProperty {
	var prop controllerProperty

	// Get controller name.
	vendorNameStr := controller.Send(selVendorName)
	if vendorNameStr != 0 {
		prop.name = nsStringToGoString(vendorNameStr)
	}
	if prop.name == "" {
		prop.name = "MFi Gamepad"
	}

	extGamepad := controller.Send(selExtendedGamepad)
	if extGamepad == 0 {
		return prop
	}

	var vendor, product, subtype uint16

	// Detect controller type via productCategory (macOS 10.15+) or vendorName.
	isXbox := false
	isPS4 := false
	isPS5 := false

	productCategory := controller.Send(selProductCategory)
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
		vendorName := controller.Send(selVendorName)
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
	if extGamepad.Send(selRespondsToSelector, selLeftThumbstickButton) != 0 && extGamepad.Send(selLeftThumbstickButton) != 0 {
		prop.buttonMask |= (1 << kControllerButtonLeftStick)
		prop.nButtons++
	}
	if extGamepad.Send(selRespondsToSelector, selRightThumbstickButton) != 0 && extGamepad.Send(selRightThumbstickButton) != 0 {
		prop.buttonMask |= (1 << kControllerButtonRightStick)
		prop.nButtons++
	}
	if extGamepad.Send(selRespondsToSelector, selButtonOptions) != 0 && extGamepad.Send(selButtonOptions) != 0 {
		prop.buttonMask |= (1 << kControllerButtonBack)
		prop.nButtons++
	}
	if extGamepad.Send(selRespondsToSelector, selButtonHome) != 0 && extGamepad.Send(selButtonHome) != 0 {
		prop.buttonMask |= (1 << kControllerButtonGuide)
		prop.nButtons++
	}

	prop.buttonMask |= (1 << kControllerButtonStart)
	prop.nButtons++

	// Physical input profile buttons (GCInputDualShockTouchpad, Xbox paddles, etc.).
	if controller.Send(selRespondsToSelector, selPhysicalInputProfile) != 0 {
		profile := controller.Send(selPhysicalInputProfile)
		if profile != 0 {
			profileButtons := profile.Send(selButtons)
			if profileButtons != 0 {
				if gcInputDualShockTouchpadButton != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputDualShockTouchpadButton) != 0 {
					prop.hasDualshockTouchpad = true
					prop.buttonMask |= (1 << kControllerButtonMisc1)
					prop.nButtons++
				}
				if gcInputXboxPaddleOne != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleOne) != 0 {
					prop.hasXboxPaddles = true
					prop.buttonMask |= (1 << kControllerButtonPaddle1)
					prop.nButtons++
				}
				if gcInputXboxPaddleTwo != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleTwo) != 0 {
					prop.hasXboxPaddles = true
					prop.buttonMask |= (1 << kControllerButtonPaddle2)
					prop.nButtons++
				}
				if gcInputXboxPaddleThree != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleThree) != 0 {
					prop.hasXboxPaddles = true
					prop.buttonMask |= (1 << kControllerButtonPaddle3)
					prop.nButtons++
				}
				if gcInputXboxPaddleFour != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleFour) != 0 {
					prop.hasXboxPaddles = true
					prop.buttonMask |= (1 << kControllerButtonPaddle4)
					prop.nButtons++
				}
				if gcInputXboxShareButton != 0 && profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxShareButton) != 0 {
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
	return objc.Send[float32](element, selValue)
}

// getIsPressed reads the boolean isPressed property from an ObjC button element.
func getIsPressed(element objc.ID) bool {
	return element.Send(selIsPressed) != 0
}

// getHatState reads the hat state from a GCControllerDirectionPad.
func getHatState(dpad objc.ID) uint8 {
	var hat uint8
	if getIsPressed(dpad.Send(selUp)) {
		hat |= kHatUp
	} else if getIsPressed(dpad.Send(selDown)) {
		hat |= kHatDown
	}
	if getIsPressed(dpad.Send(selLeft)) {
		hat |= kHatLeft
	} else if getIsPressed(dpad.Send(selRight)) {
		hat |= kHatRight
	}
	return hat
}

// getControllerStateGC reads the current input state from a GCController.
func getControllerStateGC(controllerPtr uintptr, buttonMask uint32, nHats int,
	hasDualshockTouchpad, hasXboxPaddles, hasXboxShareButton bool) controllerState {

	controller := objc.ID(controllerPtr)
	var state controllerState

	extGamepad := controller.Send(selExtendedGamepad)
	if extGamepad == 0 {
		return state
	}

	// Axes.
	leftStick := extGamepad.Send(selLeftThumbstick)
	rightStick := extGamepad.Send(selRightThumbstick)
	state.axes[0] = getAxisValue(leftStick.Send(selXAxis))
	state.axes[1] = -getAxisValue(leftStick.Send(selYAxis))
	state.axes[2] = getAxisValue(extGamepad.Send(selLeftTrigger))*2 - 1
	state.axes[3] = getAxisValue(rightStick.Send(selXAxis))
	state.axes[4] = -getAxisValue(rightStick.Send(selYAxis))
	state.axes[5] = getAxisValue(extGamepad.Send(selRightTrigger))*2 - 1

	// Buttons.
	buttonCount := 0
	setButton := func(pressed bool) {
		if pressed {
			state.buttons[buttonCount] = 1
		}
		buttonCount++
	}
	setButton(getIsPressed(extGamepad.Send(selButtonA)))
	setButton(getIsPressed(extGamepad.Send(selButtonB)))
	setButton(getIsPressed(extGamepad.Send(selButtonX)))
	setButton(getIsPressed(extGamepad.Send(selButtonY)))
	setButton(getIsPressed(extGamepad.Send(selLeftShoulder)))
	setButton(getIsPressed(extGamepad.Send(selRightShoulder)))

	if buttonMask&(1<<kControllerButtonLeftStick) != 0 {
		setButton(getIsPressed(extGamepad.Send(selLeftThumbstickButton)))
	}
	if buttonMask&(1<<kControllerButtonRightStick) != 0 {
		setButton(getIsPressed(extGamepad.Send(selRightThumbstickButton)))
	}
	if buttonMask&(1<<kControllerButtonBack) != 0 {
		setButton(getIsPressed(extGamepad.Send(selButtonOptions)))
	}
	if buttonMask&(1<<kControllerButtonGuide) != 0 {
		setButton(getIsPressed(extGamepad.Send(selButtonHome)))
	}
	if buttonMask&(1<<kControllerButtonStart) != 0 {
		setButton(getIsPressed(extGamepad.Send(selButtonMenu)))
	}

	if hasDualshockTouchpad {
		profile := controller.Send(selPhysicalInputProfile)
		profileButtons := profile.Send(selButtons)
		btn := profileButtons.Send(selObjectForKeyedSubscript, gcInputDualShockTouchpadButton)
		setButton(getIsPressed(btn))
	}
	if hasXboxPaddles {
		profile := controller.Send(selPhysicalInputProfile)
		profileButtons := profile.Send(selButtons)
		if buttonMask&(1<<kControllerButtonPaddle1) != 0 {
			setButton(getIsPressed(profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleOne)))
		}
		if buttonMask&(1<<kControllerButtonPaddle2) != 0 {
			setButton(getIsPressed(profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleTwo)))
		}
		if buttonMask&(1<<kControllerButtonPaddle3) != 0 {
			setButton(getIsPressed(profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleThree)))
		}
		if buttonMask&(1<<kControllerButtonPaddle4) != 0 {
			setButton(getIsPressed(profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxPaddleFour)))
		}
	}
	if hasXboxShareButton {
		profile := controller.Send(selPhysicalInputProfile)
		profileButtons := profile.Send(selButtons)
		setButton(getIsPressed(profileButtons.Send(selObjectForKeyedSubscript, gcInputXboxShareButton)))
	}

	// Hat.
	if nHats > 0 {
		state.hat = getHatState(extGamepad.Send(selDpad))
	}

	return state
}

// addController adds a GCController to the gamepad list.
func addController(controller objc.ID) {
	// Ignore if the controller is not an actual controller (e.g., Siri Remote).
	if controller.Send(selExtendedGamepad) == 0 && controller.Send(selMicroGamepad) != 0 {
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
	if classGCController == 0 {
		return
	}

	// Add all currently connected controllers.
	controllers := objc.ID(classGCController).Send(selControllers)
	count := int(controllers.Send(selCount))
	for i := range count {
		controller := controllers.Send(selObjectAtIndex, i)
		addController(controller)
	}

	// Register for connect/disconnect notifications.
	center := objc.ID(classNSNotificationCtr).Send(selDefaultCenter)

	connectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
		controller := notification.Send(selObject)
		addController(controller)
	})

	disconnectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
		controller := notification.Send(selObject)
		removeController(controller)
	})

	// The notification name symbols are pointers to NSString* — dereference them.
	if gcControllerDidConnectNotification != 0 {
		connectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidConnectNotification))
		center.Send(selAddObserver, connectName, uintptr(0), uintptr(0), connectBlock)
	}
	if gcControllerDidDisconnectNotification != 0 {
		disconnectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidDisconnectNotification))
		center.Send(selAddObserver, disconnectName, uintptr(0), uintptr(0), disconnectBlock)
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

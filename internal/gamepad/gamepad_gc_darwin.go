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
	"encoding/hex"
	"runtime"
	"slices"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/cstrings"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
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
	hasDualShockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool
	touchSlots           [gcTouchSlotMax]gcTouchSlotKind
}

// controllerState holds the current input state of a controller.
type controllerState struct {
	buttons [32]uint8
	axes    [32]float32
	hat     uint8
}

// gcControllerToAdd is a controller waiting to be registered, along with the properties read from it
// when it was enumerated or its connect notification arrived.
type gcControllerToAdd struct {
	controller uintptr
	prop       controllerProperty
	rejected   bool
}

type gcHIDDeviceLookup struct {
	hidDeviceRegistryIDs []uint64
	// lookupComplete reports whether the HID device lookup needs no further retries.
	lookupComplete bool
}

type nativeGamepadsGC struct {
	// controllersToAdd and controllersToRemove hold one reference per entry, which update releases or
	// hands over to the gamepad.
	controllersToAdd    []gcControllerToAdd
	controllersToRemove []uintptr
	controllersMu       sync.Mutex

	// rejectedControllerHIDLookups maps rejected GC controllers to their HID device lookup results,
	// holding one reference per controller until disconnection.
	rejectedControllerHIDLookups map[uintptr]gcHIDDeviceLookup
}

// theGCGamepads is the running GameController backend. The notification blocks reference it
// directly, as theGamepads.native is a composite backend rather than this one.
var theGCGamepads *nativeGamepadsGC

func newNativeGamepadsGC() nativeGamepads {
	return &nativeGamepadsGC{}
}

func (g *nativeGamepadsGC) init(gamepads *gamepads) error {
	theGCGamepads = g

	initializeGCGamepads()
	return nil
}

func (g *nativeGamepadsGC) update(gamepads *gamepads) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	g.controllersMu.Lock()
	defer g.controllersMu.Unlock()

	for _, c := range g.controllersToAdd {
		if c.rejected {
			if _, ok := g.rejectedControllerHIDLookups[c.controller]; ok {
				objc.ID(c.controller).Send(sel_release)
				continue
			}
			if g.rejectedControllerHIDLookups == nil {
				g.rejectedControllerHIDLookups = map[uintptr]gcHIDDeviceLookup{}
			}
			g.rejectedControllerHIDLookups[c.controller] = gcHIDDeviceLookup{}
			continue
		}
		gamepads.addGCGamepad(c.controller, c.prop)
	}
	for _, controller := range g.controllersToRemove {
		if _, ok := g.rejectedControllerHIDLookups[controller]; ok {
			delete(g.rejectedControllerHIDLookups, controller)
			// Release the reference retained by addController and held by rejectedControllerHIDLookups.
			objc.ID(controller).Send(sel_release)
		}
		gamepads.removeGCGamepad(controller)
		// Release the separate reference retained by removeController for the removal queue.
		objc.ID(controller).Send(sel_release)
	}
	g.controllersToAdd = g.controllersToAdd[:0]
	g.controllersToRemove = g.controllersToRemove[:0]
	for controller, rejected := range g.rejectedControllerHIDLookups {
		if !rejected.lookupComplete {
			rejected.hidDeviceRegistryIDs, rejected.lookupComplete = gcHIDDeviceRegistryIDs(objc.ID(controller))
			g.rejectedControllerHIDLookups[controller] = rejected
		}
	}
	return nil
}

type nativeGamepadGC struct {
	controller           uintptr
	buttonMask           uint32
	hasDualShockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool
	touchSlots           [gcTouchSlotMax]gcTouchSlotKind
	leftMotor            *rumbleMotor
	rightMotor           *rumbleMotor
	vibEnd               time.Time
	cleanup              runtime.Cleanup

	axes    []float64
	buttons []bool
	hats    []int

	// touchMu guards touchTracker, which the finger elements' handlers write on the framework's
	// handler queue and the update reads.
	touchMu      sync.Mutex
	touchTracker touchSlotTracker

	// touchElements is the finger elements the handlers are set on, cleared by close.
	touchElements []gcTouchElement

	// touches is the update's snapshot of the touch surface's finger slots, one per element found
	// in the profile; it is empty for a controller without a touch surface.
	touches []touchContact
}

// close releases g's native resources. close can be called multiple times.
func (g *nativeGamepadGC) close() {
	g.cleanup.Stop()
	g.stopTouchTracking()
	releaseGCRumbleMotor(g.leftMotor)
	releaseGCRumbleMotor(g.rightMotor)
	g.leftMotor = nil
	g.rightMotor = nil
	if g.controller != 0 {
		objc.ID(g.controller).Send(sel_release)
		g.controller = 0
	}
}

func (g *nativeGamepadGC) update(gamepad *gamepads) error {
	// The extendedGamepad and physicalInputProfile getters return autoreleased objects, and the
	// gamepad update does not run inside an autorelease pool. The pool is safe here only because the
	// update goroutine is locked to an OS thread.
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	g.updateGCGamepad()
	if !g.vibEnd.IsZero() && time.Since(g.vibEnd) >= 0 {
		vibrateGCGamepad(g.leftMotor, g.rightMotor, 0, 0)
		g.vibEnd = time.Time{}
	}
	return nil
}

func (*nativeGamepadGC) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadGC) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadGC) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadGC) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepadGC) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepadGC) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadGC) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadGC) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadGC) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadGC) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadGC) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return 0
	}
	return g.hats[hat]
}

func (g *nativeGamepadGC) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	strongMagnitude = mathutil.Clamp01(strongMagnitude)
	weakMagnitude = mathutil.Clamp01(weakMagnitude)

	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		g.vibEnd = time.Time{}
		vibrateGCGamepad(g.leftMotor, g.rightMotor, 0, 0)
		return
	}
	g.vibEnd = time.Now().Add(duration)
	vibrateGCGamepad(g.leftMotor, g.rightMotor, strongMagnitude, weakMagnitude)
}

func (g *nativeGamepadGC) isVibrationAvailable() bool {
	return g.leftMotor != nil || g.rightMotor != nil
}

// isKnownRejectedHIDDevice reports whether the device is known to belong to a rejected GC controller.
func (g *nativeGamepadsGC) isKnownRejectedHIDDevice(id uint64) bool {
	for _, rejected := range g.rejectedControllerHIDLookups {
		if slices.Contains(rejected.hidDeviceRegistryIDs, id) {
			return true
		}
	}
	return false
}

// getControllerPropertyFromController extracts controller properties via ObjC.
func getControllerPropertyFromController(controller objc.ID) controllerProperty {
	var prop controllerProperty

	// Get controller name.
	prop.name = cstrings.NSStringToString(controller.Send(sel_vendorName))
	if prop.name == "" {
		prop.name = "MFi Gamepad"
	}

	var vendor, product, subtype uint16

	extGamepad := controller.Send(sel_extendedGamepad)
	if extGamepad != 0 {
		// Detect controller type via productCategory (macOS 10.15+) or vendorName.
		var isXbox bool
		var isPS4 bool
		var isPS5 bool

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
						prop.hasDualShockTouchpad = true
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
				prop.touchSlots = discoverGCTouchSlots(profile)
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
			if prop.hasDualShockTouchpad {
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
	hasDualShockTouchpad, hasXboxPaddles, hasXboxShareButton bool) controllerState {

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
	var buttonCount int
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

	if hasDualShockTouchpad {
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

// addController queues a GCController to be registered by the next update. The properties are read
// here, while the controller is known to be alive. The controller is only guaranteed to stay alive
// during the notification, so the queued entry takes a reference.
func addController(controller objc.ID) {
	rejected := controller.Send(sel_extendedGamepad) == 0
	if rejected && runtime.GOOS == "ios" {
		return
	}

	var prop controllerProperty
	if !rejected {
		prop = getControllerPropertyFromController(controller)
	}

	theGCGamepads.controllersMu.Lock()
	defer theGCGamepads.controllersMu.Unlock()

	controller.Send(sel_retain)
	theGCGamepads.controllersToAdd = append(theGCGamepads.controllersToAdd, gcControllerToAdd{
		controller: uintptr(controller),
		prop:       prop,
		rejected:   rejected,
	})
}

// removeController queues a GCController to be unregistered by the next update. The queued entry
// takes a reference.
func removeController(controller objc.ID) {
	theGCGamepads.controllersMu.Lock()
	defer theGCGamepads.controllersMu.Unlock()

	// The update only compares the entry with the gamepads by identity, but the reference keeps the
	// address from being reused by a controller queued for registration before the removal is
	// applied, which would make the comparison match the new controller's gamepad.
	controller.Send(sel_retain)
	theGCGamepads.controllersToRemove = append(theGCGamepads.controllersToRemove, uintptr(controller))
}

// addGCGamepad adds a GameController gamepad to the gamepad list, or leaves the list as it is if the
// controller is already in it. addGCGamepad consumes the caller's reference to controller. g.m must
// be held.
func (g *gamepads) addGCGamepad(controller uintptr, prop controllerProperty) {
	// A controller connected during initialization is queued twice: by the enumeration and by its
	// connect notification.
	if g.find(func(gamepad *Gamepad) bool {
		gc, ok := gamepad.native.(*nativeGamepadGC)
		return ok && gc.controller == controller
	}) != nil {
		objc.ID(controller).Send(sel_release)
		return
	}

	sdlID := hex.EncodeToString(prop.guid[:])
	gp := g.add(prop.name, sdlID)
	n := &nativeGamepadGC{
		controller:           controller,
		axes:                 make([]float64, prop.nAxes),
		buttons:              make([]bool, prop.nButtons+prop.nHats*4),
		hats:                 make([]int, prop.nHats),
		buttonMask:           prop.buttonMask,
		hasDualShockTouchpad: prop.hasDualShockTouchpad,
		hasXboxPaddles:       prop.hasXboxPaddles,
		hasXboxShareButton:   prop.hasXboxShareButton,
		touchSlots:           prop.touchSlots,
		touches:              make([]touchContact, gcTouchSlotCount(prop.touchSlots)),
		leftMotor:            createGCRumbleMotor(controller, 0),
		rightMotor:           createGCRumbleMotor(controller, 1),
	}
	gp.native = n
	n.startTouchTracking()
	n.cleanup = runtime.AddCleanup(gp, func(n *nativeGamepadGC) {
		n.close()
	}, n)
}

// removeGCGamepad removes the GameController gamepads for controller from the gamepad list. g.m must
// be held.
func (g *gamepads) removeGCGamepad(controller uintptr) {
	for {
		gp := g.find(func(gamepad *Gamepad) bool {
			gc, ok := gamepad.native.(*nativeGamepadGC)
			return ok && gc.controller == controller
		})
		if gp == nil {
			break
		}
		// Lock the gamepad so the close cannot race with a concurrent Vibrate using the motors.
		gp.close()
		g.remove(func(gamepad *Gamepad) bool {
			return gamepad == gp
		})
	}
}

func initializeGCGamepads() {
	if class_GCController == 0 {
		return
	}

	// Queue all currently connected controllers. initializeGCGamepads is called from gamepads.update
	// with theGamepads.m held, so the gamepad list must not be touched until the drain in
	// nativeGamepadsGC.update, which the same gamepads.update call reaches right after.
	controllers := objc.ID(class_GCController).Send(sel_controllers)
	count := int(controllers.Send(sel_count))
	for i := range count {
		controller := controllers.Send(sel_objectAtIndex, i)
		addController(controller)
	}

	// Register for connect/disconnect notifications.
	center := objc.ID(class_NSNotificationCenter).Send(sel_defaultCenter)

	// The notification name symbols are pointers to NSString* — dereference them.
	if gcControllerDidConnectNotification != 0 {
		connectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
			controller := notification.Send(sel_object)
			addController(controller)
		})
		// The notification center retains its own copy of the block.
		defer connectBlock.Release()

		connectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidConnectNotification))
		center.Send(sel_addObserverForName_object_queue_usingBlock, connectName, uintptr(0), uintptr(0), connectBlock)
	}
	if gcControllerDidDisconnectNotification != 0 {
		disconnectBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
			controller := notification.Send(sel_object)
			removeController(controller)
		})
		defer disconnectBlock.Release()

		disconnectName := *(*objc.ID)(unsafe.Pointer(gcControllerDidDisconnectNotification))
		center.Send(sel_addObserverForName_object_queue_usingBlock, disconnectName, uintptr(0), uintptr(0), disconnectBlock)
	}
}

func (g *nativeGamepadGC) updateGCGamepad() {
	state := getControllerStateGC(g.controller, g.buttonMask, len(g.hats),
		g.hasDualShockTouchpad, g.hasXboxPaddles, g.hasXboxShareButton)

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

	g.updateTouches()
}

// gcHIDDeviceRegistryIDs returns the registry IDs of a controller's underlying HID devices
// and whether no further lookup retries are needed. Incomplete lookups return nil, false.
func gcHIDDeviceRegistryIDs(controller objc.ID) ([]uint64, bool) {
	if _IOHIDServiceClientGetRegistryID == nil {
		return nil, true
	}
	// hidServices and service are private selectors used by GameController's HID backend.
	// Missing selectors leave ownership with GameController because the device cannot be identified.
	hidServicesSelector := objc.RegisterName("hidServices")
	serviceSelector := objc.RegisterName("service")
	if controller.Send(sel_respondsToSelector, hidServicesSelector) == 0 {
		return nil, true
	}
	services := controller.Send(hidServicesSelector)
	count := int(services.Send(sel_count))
	if count == 0 {
		return nil, false
	}
	var ids []uint64
	for i := range count {
		info := services.Send(sel_objectAtIndex, i)
		if info == 0 {
			return nil, false
		}
		if info.Send(sel_respondsToSelector, serviceSelector) == 0 {
			continue
		}
		service := info.Send(serviceSelector)
		if service == 0 {
			return nil, false
		}
		registryID := _IOHIDServiceClientGetRegistryID(uintptr(service))
		if registryID == 0 {
			return nil, false
		}
		serviceID := objc.Send[uint64](objc.ID(registryID), objc.RegisterName("unsignedLongLongValue"))
		id := hidDeviceRegistryIDForService(serviceID)
		if id == 0 {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

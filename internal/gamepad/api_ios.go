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

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation -framework GameController
//
// #import <GameController/GameController.h>
//
// static NSString* GCInputXboxShareButton = @"Button Share";
//
// enum ControllerButton {
//   kControllerButtonInvalid = -1,
//   kControllerButtonA,
//   kControllerButtonB,
//   kControllerButtonX,
//   kControllerButtonY,
//   kControllerButtonBack,
//   kControllerButtonGuide,
//   kControllerButtonStart,
//   kControllerButtonLeftStick,
//   kControllerButtonRightStick,
//   kControllerButtonLeftShoulder,
//   kControllerButtonRightShoulder,
//   kControllerButtonDpadUp,
//   kControllerButtonDpadDown,
//   kControllerButtonDpadLeft,
//   kControllerButtonDpadRight,
//   kControllerButtonMisc1,
//   kControllerButtonPaddle1,
//   kControllerButtonPaddle2,
//   kControllerButtonPaddle3,
//   kControllerButtonPaddle4,
//   kControllerButtonTouchpad,
//   kControllerButtonMax,
// };
//
// enum HatState : uint8_t {
//   kHatCentered  = 0x00,
//   kHatUp        = 0x01,
//   kHatRight     = 0x02,
//   kHatDown      = 0x04,
//   kHatLeft      = 0x08,
//   kHatRightUp   = kHatRight | kHatUp,
//   kHatRightDown = kHatRight | kHatDown,
//   kHatLeftUp    = kHatLeft | kHatUp,
//   kHatLeftDown  = kHatLeft | kHatDown,
// };
//
// enum USBVendor {
//   kUSBVendorApple     = 0x05ac,
//   kUSBVendorMicrosoft = 0x045e,
//   kUSBVendorSony      = 0x054c,
// };
//
// enum USBProduct {
//   kUSBProductSonyDS4Slim                  = 0x09cc,
//   kUSBProductSonyDS5                      = 0x0ce6,
//   kUSBProductXboxOneEliteSeries2Bluetooth = 0x0b05,
//   kUSBProductXboxOneSRev1Bluetooth        = 0x02e0,
//   kUSBProductXboxSeriesXBluetooth         = 0x0b13,
// };
//
// struct ControllerProperty {
//   uint8_t nAxes;
//   uint8_t nButtons;
//   uint8_t nHats;
//   uint16_t buttonMask;
//   char guid[16];
//   char name[256];
//   bool hasDualshockTouchpad;
//   bool hasXboxPaddles;
//   bool hasXboxShareButton;
// };
//
// void ebitenAddGamepad(uintptr_t controller, struct ControllerProperty* prop);
// void ebitenRemoveGamepad(uintptr_t controller);
//
// static size_t min(size_t a, size_t b) {
//   return a < b ? a : b;
// }
//
// static void getControllerPropertyFromController(GCController* controller, struct ControllerProperty* property) {
//   @autoreleasepool {
//     uint16_t vendor = 0;
//     uint16_t product = 0;
//     uint16_t subtype = 0;
//
//     const char* name = nil;
//     if (controller.vendorName) {
//       name = controller.vendorName.UTF8String;
//     }
//     if (!name) {
//       name = "MFi Gamepad";
//     }
//     memcpy(property->name, name, min(sizeof(property->name), strlen(name)));
//
//     if (controller.extendedGamepad) {
//       GCExtendedGamepad* gamepad = controller.extendedGamepad;
//
//       bool isXbox = false;
//       bool isPS4 = false;
//       bool isPS5 = false;
//       if (@available(macOS 10.15, iOS 13.0, tvOS 13.0, *)) {
//         NSString* productCategory = [controller productCategory];
//         if ([productCategory isEqualToString:@"DualShock 4"]) {
//           isPS4 = 1;
//         } else if ([productCategory isEqualToString:@"DualSense"]) {
//           isPS5 = 1;
//         } else if ([productCategory isEqualToString:@"Xbox One"]) {
//           isXbox = 1;
//         }
//       } else {
//         NSString* vendorName = [controller vendorName];
//         if ([vendorName isEqualToString:@"DUALSHOCK"]) {
//           isPS4 = 1;
//         } else if ([vendorName isEqualToString:@"DualSense"]) {
//           isPS5 = 1;
//         } else if ([vendorName isEqualToString:@"Xbox"]) {
//           isXbox = 1;
//         }
//       }
//
//       property->buttonMask |= (1 << kControllerButtonA);
//       property->buttonMask |= (1 << kControllerButtonB);
//       property->buttonMask |= (1 << kControllerButtonX);
//       property->buttonMask |= (1 << kControllerButtonY);
//       property->buttonMask |= (1 << kControllerButtonLeftShoulder);
//       property->buttonMask |= (1 << kControllerButtonRightShoulder);
//       property->nButtons += 6;
//
// #pragma clang diagnostic push
// #pragma clang diagnostic ignored "-Wunguarded-availability-new"
//
//       if ([gamepad respondsToSelector:@selector(leftThumbstickButton)] && gamepad.leftThumbstickButton) {
//         property->buttonMask |= (1 << kControllerButtonLeftStick);
//         property->nButtons++;
//       }
//       if ([gamepad respondsToSelector:@selector(rightThumbstickButton)] && gamepad.rightThumbstickButton) {
//         property->buttonMask |= (1 << kControllerButtonRightStick);
//         property->nButtons++;
//       }
//       if ([gamepad respondsToSelector:@selector(buttonOptions)] && gamepad.buttonOptions) {
//         property->buttonMask |= (1 << kControllerButtonBack);
//         property->nButtons++;
//       }
//       if ([gamepad respondsToSelector:@selector(buttonHome)] && gamepad.buttonHome) {
//         property->buttonMask |= (1 << kControllerButtonGuide);
//         property->nButtons++;
//       }
//
//       property->buttonMask |= (1 << kControllerButtonStart);
//       property->nButtons++;
//
//       if ([controller respondsToSelector:@selector(physicalInputProfile)]) {
//         if (controller.physicalInputProfile.buttons[GCInputDualShockTouchpadButton] != nil) {
//           property->hasDualshockTouchpad = true;
//           property->buttonMask |= (1 << kControllerButtonMisc1);
//           property->nButtons++;
//         }
//         if (controller.physicalInputProfile.buttons[GCInputXboxPaddleOne] != nil) {
//           property->hasXboxPaddles = true;
//           property->buttonMask |= (1 << kControllerButtonPaddle1);
//           property->nButtons++;
//         }
//         if (controller.physicalInputProfile.buttons[GCInputXboxPaddleTwo] != nil) {
//           property->hasXboxPaddles = true;
//           property->buttonMask |= (1 << kControllerButtonPaddle2);
//           property->nButtons++;
//         }
//         if (controller.physicalInputProfile.buttons[GCInputXboxPaddleThree] != nil) {
//           property->hasXboxPaddles = true;
//           property->buttonMask |= (1 << kControllerButtonPaddle3);
//           property->nButtons++;
//         }
//         if (controller.physicalInputProfile.buttons[GCInputXboxPaddleFour] != nil) {
//           property->hasXboxPaddles = true;
//           property->buttonMask |= (1 << kControllerButtonPaddle4);
//           property->nButtons++;
//         }
//         if (controller.physicalInputProfile.buttons[GCInputXboxShareButton] != nil) {
//           property->hasXboxShareButton = true;
//           property->buttonMask |= (1 << kControllerButtonMisc1);
//           property->nButtons++;
//         }
//       }
//
// #pragma clang diagnostic pop
//
//       if (isXbox) {
//         vendor = kUSBVendorMicrosoft;
//         if (property->hasXboxPaddles) {
//           product = kUSBProductXboxOneEliteSeries2Bluetooth;
//           subtype = 1;
//         } else if (property->hasXboxShareButton) {
//           product = kUSBProductXboxSeriesXBluetooth;
//           subtype = 1;
//         } else {
//           product = kUSBProductXboxOneSRev1Bluetooth;
//           subtype = 0;
//         }
//       } else if (isPS4) {
//         vendor = kUSBVendorSony;
//         product = kUSBProductSonyDS4Slim;
//         if (property->hasDualshockTouchpad) {
//           subtype = 1;
//         } else {
//           subtype = 0;
//         }
//       } else if (isPS5) {
//         vendor = kUSBVendorSony;
//         product = kUSBProductSonyDS5;
//         subtype = 0;
//       } else {
//         vendor = kUSBVendorApple;
//         product = 1;
//         subtype = 1;
//       }
//
//       property->nAxes = 6;
//       property->nHats = 1;
//     }
//
//     const int kSDLHardwareBusBluetooth = 0x05;
//     property->guid[0] = (uint8_t)(kSDLHardwareBusBluetooth);
//     property->guid[1] = (uint8_t)(kSDLHardwareBusBluetooth >> 8);
//     property->guid[2] = 0;
//     property->guid[3] = 0;
//     property->guid[4] = (uint8_t)(vendor);
//     property->guid[5] = (uint8_t)(vendor >> 8);
//     property->guid[6] = 0;
//     property->guid[7] = 0;
//     property->guid[8] = (uint8_t)(product);
//     property->guid[9] = (uint8_t)(product >> 8);
//     property->guid[10] = 0;
//     property->guid[11] = 0;
//     property->guid[12] = (uint8_t)(property->buttonMask);
//     property->guid[13] = (uint8_t)(property->buttonMask >> 8);
//     if (vendor == kUSBVendorApple) {
//       property->guid[14] = 'm';
//     } else {
//       property->guid[14] = 0;
//     }
//     property->guid[15] = subtype;
//   }
// }
//
// static void addController(GCController* controller) {
//   // Ignore if the controller is not an actual controller.
//   if (!controller.extendedGamepad && controller.microGamepad) {
//     return;
//   }
//
//   struct ControllerProperty property = {};
//   getControllerPropertyFromController(controller, &property);
//   ebitenAddGamepad((uintptr_t)(controller), &property);
// }
//
// static void removeController(GCController* controller) {
//   ebitenRemoveGamepad((uintptr_t)(controller));
// }
//
// struct ControllerState {
//   uint8_t buttons[32];
//   float axes[32];
//   enum HatState hat;
// };
//
// static enum HatState getHatState(GCControllerDirectionPad* dpad) {
//   enum HatState hat = 0;
//   if (dpad.up.isPressed) {
//     hat |= kHatUp;
//   } else if (dpad.down.isPressed) {
//     hat |= kHatDown;
//   }
//   if (dpad.left.isPressed) {
//     hat |= kHatLeft;
//   } else if (dpad.right.isPressed) {
//     hat |= kHatRight;
//   }
//   return hat;
// }
//
// static void getControllerState(uintptr_t controller_ptr, struct ControllerState* controllerState,
//                                uint16_t buttonMask, uint8_t nHats,
//                                bool hasDualshockTouchpad, bool hasXboxPaddles, bool hasXboxShareButton) {
//   GCController* controller = (GCController*)(controller_ptr);
//   @autoreleasepool {
//     if (controller.extendedGamepad) {
//       GCExtendedGamepad* gamepad = controller.extendedGamepad;
//
//       controllerState->axes[0] = gamepad.leftThumbstick.xAxis.value;
//       controllerState->axes[1] = -gamepad.leftThumbstick.yAxis.value;
//       controllerState->axes[2] = gamepad.leftTrigger.value * 2 - 1;
//       controllerState->axes[3] = gamepad.rightThumbstick.xAxis.value;
//       controllerState->axes[4] = -gamepad.rightThumbstick.yAxis.value;
//       controllerState->axes[5] = gamepad.rightTrigger.value * 2 - 1;
//
//       int buttonCount = 0;
//       controllerState->buttons[buttonCount++] = gamepad.buttonA.isPressed;
//       controllerState->buttons[buttonCount++] = gamepad.buttonB.isPressed;
//       controllerState->buttons[buttonCount++] = gamepad.buttonX.isPressed;
//       controllerState->buttons[buttonCount++] = gamepad.buttonY.isPressed;
//       controllerState->buttons[buttonCount++] = gamepad.leftShoulder.isPressed;
//       controllerState->buttons[buttonCount++] = gamepad.rightShoulder.isPressed;
//
// #pragma clang diagnostic push
// #pragma clang diagnostic ignored "-Wunguarded-availability-new"
//
//       if (buttonMask & (1 << kControllerButtonLeftStick)) {
//         controllerState->buttons[buttonCount++] = gamepad.leftThumbstickButton.isPressed;
//       }
//       if (buttonMask & (1 << kControllerButtonRightStick)) {
//         controllerState->buttons[buttonCount++] = gamepad.rightThumbstickButton.isPressed;
//       }
//       if (buttonMask & (1 << kControllerButtonBack)) {
//         controllerState->buttons[buttonCount++] = gamepad.buttonOptions.isPressed;
//       }
//       if (buttonMask & (1 << kControllerButtonGuide)) {
//         controllerState->buttons[buttonCount++] = gamepad.buttonHome.isPressed;
//       }
//       if (buttonMask & (1 << kControllerButtonStart)) {
//         controllerState->buttons[buttonCount++] = gamepad.buttonMenu.isPressed;
//       }
//
//       if (hasDualshockTouchpad) {
//         controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputDualShockTouchpadButton].isPressed;
//       }
//       if (hasXboxPaddles) {
//         if (buttonMask & (1 << kControllerButtonPaddle1)) {
//           controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputXboxPaddleOne].isPressed;
//         }
//         if (buttonMask & (1 << kControllerButtonPaddle2)) {
//           controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputXboxPaddleTwo].isPressed;
//         }
//         if (buttonMask & (1 << kControllerButtonPaddle3)) {
//           controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputXboxPaddleThree].isPressed;
//         }
//         if (buttonMask & (1 << kControllerButtonPaddle4)) {
//           controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputXboxPaddleFour].isPressed;
//         }
//       }
//       if (hasXboxShareButton) {
//         controllerState->buttons[buttonCount++] = controller.physicalInputProfile.buttons[GCInputXboxShareButton].isPressed;
//       }
//
// #pragma clang diagnostic pop
//
//       if (nHats) {
//         controllerState->hat = getHatState(gamepad.dpad);
//       }
//     }
//   }
// }
//
// static void initializeGamepads(void) {
//   @autoreleasepool {
//     for (GCController* controller in [GCController controllers]) {
//       addController(controller);
//     }
//     NSNotificationCenter* center = [NSNotificationCenter defaultCenter];
//     [center addObserverForName:GCControllerDidConnectNotification
//                         object:nil
//                          queue:nil
//                     usingBlock:^(NSNotification* notification) {
//                                  addController(notification.object);
//                                }];
//     [center addObserverForName:GCControllerDidDisconnectNotification
//                         object:nil
//                          queue:nil
//                     usingBlock:^(NSNotification* notification) {
//                                  removeController(notification.object);
//                                }];
//   }
// }
import "C"

import (
	"encoding/hex"
	"unsafe"
)

//export ebitenAddGamepad
func ebitenAddGamepad(controller C.uintptr_t, prop *C.struct_ControllerProperty) {
	theGamepads.addIOSGamepad(controller, prop)
}

//export ebitenRemoveGamepad
func ebitenRemoveGamepad(controller C.uintptr_t) {
	theGamepads.removeIOSGamepad(controller)
}

func (g *gamepads) addIOSGamepad(controller C.uintptr_t, prop *C.struct_ControllerProperty) {
	g.m.Lock()
	defer g.m.Unlock()

	name := C.GoString(&prop.name[0])
	sdlID := hex.EncodeToString(C.GoBytes(unsafe.Pointer(&prop.guid[0]), 16))
	gp := g.add(name, sdlID)
	gp.native = &nativeGamepadImpl{
		controller:           uintptr(controller),
		axes:                 make([]float64, prop.nAxes),
		buttons:              make([]bool, prop.nButtons+prop.nHats*4),
		hats:                 make([]int, prop.nHats),
		buttonMask:           uint16(prop.buttonMask),
		hasDualshockTouchpad: bool(prop.hasDualshockTouchpad),
		hasXboxPaddles:       bool(prop.hasXboxPaddles),
		hasXboxShareButton:   bool(prop.hasXboxShareButton),
	}
}

func (g *gamepads) removeIOSGamepad(controller C.uintptr_t) {
	g.m.Lock()
	defer g.m.Unlock()

	g.remove(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).controller == uintptr(controller)
	})
}

func initializeIOSGamepads() {
	C.initializeGamepads()
}

func (g *nativeGamepadImpl) updateIOSGamepad() {
	var state C.struct_ControllerState
	C.getControllerState(C.uintptr_t(g.controller), &state, C.uint16_t(g.buttonMask), C.uint8_t(len(g.hats)),
		C.bool(g.hasDualshockTouchpad), C.bool(g.hasXboxPaddles), C.bool(g.hasXboxShareButton))

	nButtons := len(g.buttons) - len(g.hats)*4
	for i := 0; i < nButtons; i++ {
		g.buttons[i] = state.buttons[i] != 0
	}

	// Follow the GLFW way to process hats.
	// See _glfwInputJoystickHat.
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

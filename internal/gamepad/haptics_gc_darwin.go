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

package gamepad

import (
	"math"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// rumbleMotor manages a CoreHaptics engine and player for vibration.
type rumbleMotor struct {
	engine objc.ID // CHHapticEngine
	player objc.ID // id<CHHapticPatternPlayer>
	active bool
}

// ObjC classes for CoreHaptics (initialized in init).
var (
	class_CHHapticEngine           objc.Class
	class_CHHapticEventParameter   objc.Class
	class_CHHapticEvent            objc.Class
	class_CHHapticPattern          objc.Class
	class_CHHapticDynamicParameter objc.Class
	class_NSArray                  objc.Class
)

// ObjC selectors for CoreHaptics.
var (
	sel_haptics                                            objc.SEL
	sel_createEngineWithLocality                           objc.SEL
	sel_startAndReturnError                                objc.SEL
	sel_stopWithCompletionHandler                          objc.SEL
	sel_createPlayerWithPattern_error                      objc.SEL
	sel_sendParameters_atTime_error                        objc.SEL
	sel_startAtTime_error                                  objc.SEL
	sel_stopAtTime_error                                   objc.SEL
	sel_supportedLocalities                                objc.SEL
	sel_containsObject                                     objc.SEL
	sel_initWithParameterID_value                          objc.SEL
	sel_initWithEventType_parameters_relativeTime_duration objc.SEL
	sel_initWithEvents_parameters_error                    objc.SEL
	sel_initWithParameterID_value_relativeTime             objc.SEL
	sel_release                                            objc.SEL
	sel_retain                                             objc.SEL
	sel_init                                               objc.SEL
	sel_arrayWithObjects_count                             objc.SEL
	sel_array                                              objc.SEL
)

// CoreHaptics string/float constants (loaded from framework symbols).
var (
	chHapticsLocalityLeftHandle  objc.ID
	chHapticsLocalityRightHandle objc.ID
	chHapticsLocalityHandles     objc.ID

	chHapticEventTypeHapticContinuous             objc.ID
	chHapticEventParameterIDHapticIntensity       objc.ID
	chHapticDynamicParameterIDHapticIntensityCtrl objc.ID

	gcHapticDurationInfinite float32
)

var coreHapticsAvailable bool

func init() {
	// Load CoreHaptics framework.
	ch, err := purego.Dlopen("/System/Library/Frameworks/CoreHaptics.framework/CoreHaptics", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}

	class_CHHapticEngine = objc.GetClass("CHHapticEngine")
	if class_CHHapticEngine == 0 {
		// CoreHaptics not available (pre-macOS 10.15).
		return
	}

	class_CHHapticEventParameter = objc.GetClass("CHHapticEventParameter")
	class_CHHapticEvent = objc.GetClass("CHHapticEvent")
	class_CHHapticPattern = objc.GetClass("CHHapticPattern")
	class_CHHapticDynamicParameter = objc.GetClass("CHHapticDynamicParameter")
	class_NSArray = objc.GetClass("NSArray")

	sel_haptics = objc.RegisterName("haptics")
	sel_createEngineWithLocality = objc.RegisterName("createEngineWithLocality:")
	sel_startAndReturnError = objc.RegisterName("startAndReturnError:")
	sel_stopWithCompletionHandler = objc.RegisterName("stopWithCompletionHandler:")
	sel_createPlayerWithPattern_error = objc.RegisterName("createPlayerWithPattern:error:")
	sel_sendParameters_atTime_error = objc.RegisterName("sendParameters:atTime:error:")
	sel_startAtTime_error = objc.RegisterName("startAtTime:error:")
	sel_stopAtTime_error = objc.RegisterName("stopAtTime:error:")
	sel_supportedLocalities = objc.RegisterName("supportedLocalities")
	sel_containsObject = objc.RegisterName("containsObject:")
	sel_initWithParameterID_value = objc.RegisterName("initWithParameterID:value:")
	sel_initWithEventType_parameters_relativeTime_duration = objc.RegisterName("initWithEventType:parameters:relativeTime:duration:")
	sel_initWithEvents_parameters_error = objc.RegisterName("initWithEvents:parameters:error:")
	sel_initWithParameterID_value_relativeTime = objc.RegisterName("initWithParameterID:value:relativeTime:")
	sel_release = objc.RegisterName("release")
	sel_retain = objc.RegisterName("retain")
	sel_init = objc.RegisterName("init")
	sel_arrayWithObjects_count = objc.RegisterName("arrayWithObjects:count:")
	sel_array = objc.RegisterName("array")

	// Load string constants from GameController framework.
	gc, err := purego.Dlopen("/System/Library/Frameworks/GameController.framework/GameController", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}

	loadNSStringConst := func(framework uintptr, name string) objc.ID {
		ptr, err := purego.Dlsym(framework, name)
		if err != nil {
			return 0
		}
		return *(*objc.ID)(unsafe.Pointer(ptr))
	}

	chHapticsLocalityLeftHandle = loadNSStringConst(gc, "GCHapticsLocalityLeftHandle")
	chHapticsLocalityRightHandle = loadNSStringConst(gc, "GCHapticsLocalityRightHandle")
	chHapticsLocalityHandles = loadNSStringConst(gc, "GCHapticsLocalityHandles")

	chHapticEventTypeHapticContinuous = loadNSStringConst(ch, "CHHapticEventTypeHapticContinuous")
	chHapticEventParameterIDHapticIntensity = loadNSStringConst(ch, "CHHapticEventParameterIDHapticIntensity")
	chHapticDynamicParameterIDHapticIntensityCtrl = loadNSStringConst(ch, "CHHapticDynamicParameterIDHapticIntensityControl")

	// Load GCHapticDurationInfinite (float constant).
	durPtr, err := purego.Dlsym(gc, "GCHapticDurationInfinite")
	if err == nil {
		gcHapticDurationInfinite = *(*float32)(unsafe.Pointer(durPtr))
	} else {
		gcHapticDurationInfinite = math.Float32frombits(0x7F800000) // +Inf
	}

	coreHapticsAvailable = true
}

func createGCRumbleMotor(controller uintptr, which int) uintptr {
	if !coreHapticsAvailable {
		return 0
	}

	controllerObj := objc.ID(controller)
	haptics := controllerObj.Send(sel_haptics)
	if haptics == 0 {
		return 0
	}

	supportedLocalities := haptics.Send(sel_supportedLocalities)
	if supportedLocalities == 0 || chHapticsLocalityHandles == 0 {
		return 0
	}
	if supportedLocalities.Send(sel_containsObject, chHapticsLocalityHandles) == 0 {
		return 0
	}

	var locality objc.ID
	if which == 0 {
		locality = chHapticsLocalityLeftHandle
	} else {
		locality = chHapticsLocalityRightHandle
	}

	engine := haptics.Send(sel_createEngineWithLocality, locality)
	if engine == 0 {
		return 0
	}

	// Start the engine.
	var nsError objc.ID
	engine.Send(sel_startAndReturnError, uintptr(unsafe.Pointer(&nsError)))
	if nsError != 0 {
		return 0
	}

	// Create a continuous haptic event with intensity 1.0.
	intensityParam := objc.ID(class_CHHapticEventParameter).Send(sel_alloc).Send(
		sel_initWithParameterID_value,
		chHapticEventParameterIDHapticIntensity,
		float32(1.0),
	)

	// Create an NSArray with the parameter.
	paramArray := makeNSArray(intensityParam)

	event := objc.ID(class_CHHapticEvent).Send(sel_alloc).Send(
		sel_initWithEventType_parameters_relativeTime_duration,
		chHapticEventTypeHapticContinuous,
		paramArray,
		float64(0),                        // relativeTime
		float64(gcHapticDurationInfinite), // duration (NSTimeInterval)
	)
	intensityParam.Send(sel_release)

	// Create pattern.
	eventArray := makeNSArray(event)
	emptyArray := objc.ID(class_NSArray).Send(sel_array)

	nsError = 0
	pattern := objc.ID(class_CHHapticPattern).Send(sel_alloc).Send(
		sel_initWithEvents_parameters_error,
		eventArray,
		emptyArray,
		uintptr(unsafe.Pointer(&nsError)),
	)
	event.Send(sel_release)
	if nsError != 0 {
		if pattern != 0 {
			pattern.Send(sel_release)
		}
		engine.Send(sel_stopWithCompletionHandler, uintptr(0))
		return 0
	}

	// Create player.
	nsError = 0
	player := engine.Send(sel_createPlayerWithPattern_error, pattern, uintptr(unsafe.Pointer(&nsError)))
	pattern.Send(sel_release)
	if nsError != 0 {
		engine.Send(sel_stopWithCompletionHandler, uintptr(0))
		return 0
	}

	motor := &rumbleMotor{
		engine: engine.Send(sel_retain),
		player: player.Send(sel_retain),
		active: false,
	}

	return uintptr(unsafe.Pointer(motor))
}

// makeNSArray creates an NSArray containing a single object.
func makeNSArray(obj objc.ID) objc.ID {
	objects := [1]uintptr{uintptr(obj)}
	return objc.ID(class_NSArray).Send(sel_arrayWithObjects_count, uintptr(unsafe.Pointer(&objects[0])), 1)
}

func releaseGCRumbleMotor(motorPtr uintptr) {
	if motorPtr == 0 {
		return
	}
	if !coreHapticsAvailable {
		return
	}

	motor := (*rumbleMotor)(unsafe.Pointer(motorPtr))
	if motor.active {
		var nsError objc.ID
		motor.player.Send(sel_stopAtTime_error, float64(0), uintptr(unsafe.Pointer(&nsError)))
	}
	motor.engine.Send(sel_stopWithCompletionHandler, uintptr(0))
	motor.player.Send(sel_release)
	motor.engine.Send(sel_release)
}

func vibrateGCGamepad(left, right uintptr, strong, weak float64) {
	// In common gamepads, the left motor emits low-frequency vibrations and the right motor emits high-frequency vibrations.
	// See also:
	// * https://learn.microsoft.com/en-us/windows/uwp/gaming/gamepad-and-vibration#using-the-vibration-motors
	// * https://docs.unity3d.com/Packages/com.unity.inputsystem@1.19/manual/Gamepad.html#rumble
	vibrateMotor(left, strong)
	vibrateMotor(right, weak)
}

func vibrateMotor(motorPtr uintptr, intensity float64) {
	if motorPtr == 0 || !coreHapticsAvailable {
		return
	}

	motor := (*rumbleMotor)(unsafe.Pointer(motorPtr))
	var nsError objc.ID

	if intensity <= 0 {
		if motor.active {
			motor.player.Send(sel_stopAtTime_error, float64(0), uintptr(unsafe.Pointer(&nsError)))
			motor.active = false
		}
	} else {
		// Create a dynamic parameter to control intensity.
		param := objc.ID(class_CHHapticDynamicParameter).Send(sel_alloc).Send(
			sel_initWithParameterID_value_relativeTime,
			chHapticDynamicParameterIDHapticIntensityCtrl,
			float32(intensity),
			float64(0), // relativeTime
		)
		paramArray := makeNSArray(param)
		motor.player.Send(sel_sendParameters_atTime_error, paramArray, float64(0), uintptr(unsafe.Pointer(&nsError)))
		param.Send(sel_release)
		if !motor.active {
			motor.player.Send(sel_startAtTime_error, float64(0), uintptr(unsafe.Pointer(&nsError)))
			motor.active = true
		}
	}
}

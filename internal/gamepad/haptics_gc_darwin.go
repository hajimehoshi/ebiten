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
	classCHHapticEngine         objc.Class
	classCHHapticEventParameter objc.Class
	classCHHapticEvent          objc.Class
	classCHHapticPattern        objc.Class
	classCHHapticDynamicParam   objc.Class
	classNSArrayCH              objc.Class
)

// ObjC selectors for CoreHaptics.
var (
	selCHHaptics                   objc.SEL
	selCHCreateEngineWithLocality  objc.SEL
	selCHStartAndReturnError       objc.SEL
	selCHStopWithCompletionHandler objc.SEL
	selCHCreatePlayerWithPattern   objc.SEL
	selCHSendParameters            objc.SEL
	selCHStartAtTime               objc.SEL
	selCHStopAtTime                objc.SEL
	selCHSupportedLocalities       objc.SEL
	selCHContainsObject            objc.SEL
	selCHInitWithParameterID       objc.SEL
	selCHInitWithEventType         objc.SEL
	selCHInitWithEvents            objc.SEL
	selCHInitDynParam              objc.SEL
	selCHAlloc                     objc.SEL
	selCHRelease                   objc.SEL
	selCHRetain                    objc.SEL
	selCHInit                      objc.SEL
	selCHArrayWithObjects          objc.SEL
	selCHArray                     objc.SEL
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

	classCHHapticEngine = objc.GetClass("CHHapticEngine")
	if classCHHapticEngine == 0 {
		// CoreHaptics not available (pre-macOS 10.15).
		return
	}

	classCHHapticEventParameter = objc.GetClass("CHHapticEventParameter")
	classCHHapticEvent = objc.GetClass("CHHapticEvent")
	classCHHapticPattern = objc.GetClass("CHHapticPattern")
	classCHHapticDynamicParam = objc.GetClass("CHHapticDynamicParameter")
	classNSArrayCH = objc.GetClass("NSArray")

	selCHHaptics = objc.RegisterName("haptics")
	selCHCreateEngineWithLocality = objc.RegisterName("createEngineWithLocality:")
	selCHStartAndReturnError = objc.RegisterName("startAndReturnError:")
	selCHStopWithCompletionHandler = objc.RegisterName("stopWithCompletionHandler:")
	selCHCreatePlayerWithPattern = objc.RegisterName("createPlayerWithPattern:error:")
	selCHSendParameters = objc.RegisterName("sendParameters:atTime:error:")
	selCHStartAtTime = objc.RegisterName("startAtTime:error:")
	selCHStopAtTime = objc.RegisterName("stopAtTime:error:")
	selCHSupportedLocalities = objc.RegisterName("supportedLocalities")
	selCHContainsObject = objc.RegisterName("containsObject:")
	selCHInitWithParameterID = objc.RegisterName("initWithParameterID:value:")
	selCHInitWithEventType = objc.RegisterName("initWithEventType:parameters:relativeTime:duration:")
	selCHInitWithEvents = objc.RegisterName("initWithEvents:parameters:error:")
	selCHInitDynParam = objc.RegisterName("initWithParameterID:value:relativeTime:")
	selCHAlloc = objc.RegisterName("alloc")
	selCHRelease = objc.RegisterName("release")
	selCHRetain = objc.RegisterName("retain")
	selCHInit = objc.RegisterName("init")
	selCHArrayWithObjects = objc.RegisterName("arrayWithObjects:count:")
	selCHArray = objc.RegisterName("array")

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
	haptics := controllerObj.Send(selCHHaptics)
	if haptics == 0 {
		return 0
	}

	supportedLocalities := haptics.Send(selCHSupportedLocalities)
	if supportedLocalities == 0 || chHapticsLocalityHandles == 0 {
		return 0
	}
	if supportedLocalities.Send(selCHContainsObject, chHapticsLocalityHandles) == 0 {
		return 0
	}

	var locality objc.ID
	if which == 0 {
		locality = chHapticsLocalityLeftHandle
	} else {
		locality = chHapticsLocalityRightHandle
	}

	engine := haptics.Send(selCHCreateEngineWithLocality, locality)
	if engine == 0 {
		return 0
	}

	// Start the engine.
	var nsError objc.ID
	engine.Send(selCHStartAndReturnError, uintptr(unsafe.Pointer(&nsError)))
	if nsError != 0 {
		return 0
	}

	// Create a continuous haptic event with intensity 1.0.
	intensityParam := objc.ID(classCHHapticEventParameter).Send(selCHAlloc).Send(
		selCHInitWithParameterID,
		chHapticEventParameterIDHapticIntensity,
		math.Float64frombits(uint64(math.Float32bits(1.0))<<32), // float -> double for objc.Send
	)

	// Create an NSArray with the parameter.
	paramArray := makeNSArray(intensityParam)

	event := objc.ID(classCHHapticEvent).Send(selCHAlloc).Send(
		selCHInitWithEventType,
		chHapticEventTypeHapticContinuous,
		paramArray,
		float64(0), // relativeTime
		math.Float64frombits(uint64(math.Float32bits(gcHapticDurationInfinite))<<32), // duration (float → double)
	)
	intensityParam.Send(selCHRelease)

	// Create pattern.
	eventArray := makeNSArray(event)
	emptyArray := objc.ID(classNSArrayCH).Send(selCHArray)

	nsError = 0
	pattern := objc.ID(classCHHapticPattern).Send(selCHAlloc).Send(
		selCHInitWithEvents,
		eventArray,
		emptyArray,
		uintptr(unsafe.Pointer(&nsError)),
	)
	event.Send(selCHRelease)
	if nsError != 0 {
		if pattern != 0 {
			pattern.Send(selCHRelease)
		}
		engine.Send(selCHStopWithCompletionHandler, uintptr(0))
		return 0
	}

	// Create player.
	nsError = 0
	player := engine.Send(selCHCreatePlayerWithPattern, pattern, uintptr(unsafe.Pointer(&nsError)))
	pattern.Send(selCHRelease)
	if nsError != 0 {
		engine.Send(selCHStopWithCompletionHandler, uintptr(0))
		return 0
	}

	motor := &rumbleMotor{
		engine: engine.Send(selCHRetain),
		player: player.Send(selCHRetain),
		active: false,
	}

	return uintptr(unsafe.Pointer(motor))
}

// makeNSArray creates an NSArray containing a single object.
func makeNSArray(obj objc.ID) objc.ID {
	objects := [1]uintptr{uintptr(obj)}
	return objc.ID(classNSArrayCH).Send(selCHArrayWithObjects, uintptr(unsafe.Pointer(&objects[0])), 1)
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
		motor.player.Send(selCHStopAtTime, float64(0), uintptr(unsafe.Pointer(&nsError)))
	}
	motor.engine.Send(selCHStopWithCompletionHandler, uintptr(0))
	motor.player.Send(selCHRelease)
	motor.engine.Send(selCHRelease)
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
			motor.player.Send(selCHStopAtTime, float64(0), uintptr(unsafe.Pointer(&nsError)))
			motor.active = false
		}
	} else {
		// Create a dynamic parameter to control intensity.
		param := objc.ID(classCHHapticDynamicParam).Send(selCHAlloc).Send(
			selCHInitDynParam,
			chHapticDynamicParameterIDHapticIntensityCtrl,
			math.Float64frombits(uint64(math.Float32bits(float32(intensity)))<<32), // float → double
			float64(0), // relativeTime
		)
		paramArray := makeNSArray(param)
		motor.player.Send(selCHSendParameters, paramArray, float64(0), uintptr(unsafe.Pointer(&nsError)))
		param.Send(selCHRelease)
		if !motor.active {
			motor.player.Send(selCHStartAtTime, float64(0), uintptr(unsafe.Pointer(&nsError)))
			motor.active = true
		}
	}
}

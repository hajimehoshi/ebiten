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

// #cgo LDFLAGS: -framework CoreHaptics
//
// #import <GameController/GameController.h>
// #import <CoreHaptics/CoreHaptics.h>
//
// API_AVAILABLE(macos(11.0), ios(14.0))
// @interface RumbleMotor : NSObject {
//   CHHapticEngine *_engine;
//   id<CHHapticPatternPlayer> _player;
//   BOOL _active;
// }
// - (instancetype)initWithController:(GCController *)controller locality:(GCHapticsLocality)locality;
// - (void)setIntensity:(float)intensity;
// @end
//
// API_AVAILABLE(macos(11.0), ios(14.0))
// @implementation RumbleMotor
//
// - (instancetype)initWithController:(GCController *)controller locality:(GCHapticsLocality)locality {
//   self = [super init];
//   if (!self) return nil;
//
//   CHHapticEngine *eng = [controller.haptics createEngineWithLocality:locality];
//   if (!eng) {
//     [self release];
//     return nil;
//   }
//
//   NSError *error = nil;
//   [eng startAndReturnError:&error];
//   if (error) {
//     [self release];
//     return nil;
//   }
//
//   CHHapticEventParameter *intensityParam = [[CHHapticEventParameter alloc]
//     initWithParameterID:CHHapticEventParameterIDHapticIntensity value:1.0];
//   CHHapticEvent *event = [[CHHapticEvent alloc]
//     initWithEventType:CHHapticEventTypeHapticContinuous
//     parameters:@[intensityParam]
//     relativeTime:0
//     duration:GCHapticDurationInfinite];
//   [intensityParam release];
//
//   CHHapticPattern *pattern = [[CHHapticPattern alloc] initWithEvents:@[event] parameters:@[] error:&error];
//   [event release];
//   if (error) {
//     [pattern release];
//     [eng stopWithCompletionHandler:nil];
//     [self release];
//     return nil;
//   }
//
//   id<CHHapticPatternPlayer> p = [eng createPlayerWithPattern:pattern error:&error];
//   [pattern release];
//   if (error) {
//     [eng stopWithCompletionHandler:nil];
//     [self release];
//     return nil;
//   }
//
//   _engine = [eng retain];
//   _player = [p retain];
//   _active = NO;
//
//   return self;
// }
//
// - (void)setIntensity:(float)intensity {
//   NSError *error = nil;
//   if (intensity <= 0) {
//     if (_active) {
//       [_player stopAtTime:0 error:&error];
//       _active = NO;
//     }
//   } else {
//     CHHapticDynamicParameter *param = [[CHHapticDynamicParameter alloc]
//       initWithParameterID:CHHapticDynamicParameterIDHapticIntensityControl
//       value:intensity
//       relativeTime:0];
//     [_player sendParameters:@[param] atTime:0 error:&error];
//     [param release];
//     if (!_active) {
//       [_player startAtTime:0 error:&error];
//       _active = YES;
//     }
//   }
// }
//
// - (void)dealloc {
//   if (_active) {
//     [_player stopAtTime:0 error:nil];
//   }
//   [_engine stopWithCompletionHandler:nil];
//   [_player release];
//   [_engine release];
//   [super dealloc];
// }
//
// @end
//
// static uintptr_t createRumbleMotor(uintptr_t controllerPtr, int which) {
//   if (@available(macOS 11.0, iOS 14.0, *)) {
//     @autoreleasepool {
//       GCController *controller = (GCController *)controllerPtr;
//       if (!controller.haptics) {
//         return 0;
//       }
//       if (![controller.haptics.supportedLocalities containsObject:GCHapticsLocalityHandles]) {
//         return 0;
//       }
//       GCHapticsLocality locality = (which == 0) ? GCHapticsLocalityLeftHandle : GCHapticsLocalityRightHandle;
//       RumbleMotor *motor = [[RumbleMotor alloc] initWithController:controller locality:locality];
//       if (!motor) {
//         return 0;
//       }
//       return (uintptr_t)motor;
//     }
//   }
//   return 0;
// }
//
// static void releaseRumbleMotor(uintptr_t motorPtr) {
//   if (motorPtr == 0) return;
//   if (@available(macOS 11.0, iOS 14.0, *)) {
//     @autoreleasepool {
//       [(RumbleMotor *)motorPtr release];
//     }
//   }
// }
//
// static void vibrateMotor(uintptr_t motorPtr, float intensity) {
//   if (motorPtr == 0) return;
//   if (@available(macOS 11.0, iOS 14.0, *)) {
//     @autoreleasepool {
//       [(RumbleMotor *)motorPtr setIntensity:intensity];
//     }
//   }
// }
import "C"

func createGCRumbleMotor(controller uintptr, which int) uintptr {
	return uintptr(C.createRumbleMotor(C.uintptr_t(controller), C.int(which)))
}

func releaseGCRumbleMotor(motor uintptr) {
	if motor == 0 {
		return
	}
	C.releaseRumbleMotor(C.uintptr_t(motor))
}

func vibrateGCMotor(motor uintptr, intensity float64) {
	C.vibrateMotor(C.uintptr_t(motor), C.float(intensity))
}

func vibrateGCGamepad(left, right uintptr, strong, weak float64) {
	if left != 0 {
		vibrateGCMotor(left, strong)
	}
	if right != 0 {
		vibrateGCMotor(right, weak)
	}
}

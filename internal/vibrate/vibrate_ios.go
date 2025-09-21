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

package vibrate

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AVFoundation -framework CoreHaptics -framework Foundation
//
// #import <AVFoundation/AVFoundation.h>
// #import <CoreHaptics/CoreHaptics.h>
// #include <dispatch/dispatch.h>
//
// static id initializeHapticEngine(void) {
//   if (@available(iOS 13.0, *)) {
//     if (!CHHapticEngine.capabilitiesForHardware.supportsHaptics) {
//       return nil;
//     }
//
//     NSError* error = nil;
//     // Specify the AVAudioSession's shared instance so that this won't affect
//     // the result of `[[AVAudioSession sharedInstance] secondaryAudioShouldBeSilencedHint]` (#1976).
//     CHHapticEngine* engine =
//         [[CHHapticEngine alloc] initWithAudioSession:[AVAudioSession sharedInstance]
//                                                error:&error];
//     if (error) {
//       NSLog(@"CHHapticEngine::initAndReturnError failed: %@", error);
//       return nil;
//     }
//
//     [engine startAndReturnError:&error];
//     if (error) {
//       NSLog(@"CHHapticEngine::startAndReturnError failed %@", error);
//       return nil;
//     }
//     return engine;
//   }
//   return nil;
// }
//
// static void vibrateOnMainThread(double duration, double intensity) {
//   if (@available(iOS 13.0, *)) {
//     static BOOL initializeHapticEngineCalled = NO;
//     static CHHapticEngine* engine = nil;
//     if (!initializeHapticEngineCalled) {
//       engine = (CHHapticEngine*)initializeHapticEngine();
//       initializeHapticEngineCalled = YES;
//     }
//     if (!engine) {
//       return;
//     }
//     @autoreleasepool {
//       NSDictionary* hapticDict = @{
//         (id<NSCopying>)(CHHapticPatternKeyPattern): @[
//           @{
//             (id<NSCopying>)(CHHapticPatternKeyEvent): @{
//               (id<NSCopying>)(CHHapticPatternKeyEventType):CHHapticEventTypeHapticContinuous,
//               (id<NSCopying>)(CHHapticPatternKeyTime):@0.0,
//               (id<NSCopying>)(CHHapticPatternKeyEventDuration):[NSNumber numberWithDouble:duration],
//               (id<NSCopying>)(CHHapticPatternKeyEventParameters):@[
//                 @{
//                   (id<NSCopying>)(CHHapticPatternKeyParameterID): CHHapticEventParameterIDHapticIntensity,
//                   (id<NSCopying>)(CHHapticPatternKeyParameterValue): [NSNumber numberWithDouble:intensity],
//                 },
//               ],
//             },
//           },
//         ],
//       };
//
//       NSError* error = nil;
//       CHHapticPattern* pattern = [[CHHapticPattern alloc] initWithDictionary:hapticDict
//                                                                        error:&error];
//       if (error) {
//         return;
//       }
//
//       id<CHHapticPatternPlayer> player = [engine createPlayerWithPattern:pattern
//                                                                    error:&error];
//       if (error) {
//         return;
//       }
//
//       [player startAtTime:0 error:&error];
//       if (error) {
//         NSLog(@"3, %@", [error localizedDescription]);
//         return;
//       }
//     }
//   }
// }
//
// #cgo noescape vibrate
// #cgo nocallback vibrate
// static void vibrate(double duration, double intensity) {
//   dispatch_async(dispatch_get_main_queue(), ^{
//     vibrateOnMainThread(duration, intensity);
//   });
// }
import "C"

import (
	"time"
)

func Vibrate(duration time.Duration, magnitude float64) {
	go func() {
		C.vibrate(C.double(float64(duration)/float64(time.Second)), C.double(magnitude))
	}()
}

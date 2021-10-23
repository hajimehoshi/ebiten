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

//go:build ios
// +build ios

package mobile

// #cgo LDFLAGS: -framework CoreHaptics
//
// #import <CoreHaptics/CoreHaptics.h>
//
// static CHHapticEngine* engine;
//
// static void initializeVibrate(void) {
//   if (!CHHapticEngine.capabilitiesForHardware.supportsHaptics) {
//     return;
//   }
//
//   NSError* error = nil;
//   engine = [[CHHapticEngine alloc] initAndReturnError:&error];
//   if (error) {
//     return;
//   }
//
//   [engine startAndReturnError:&error];
//   if (error) {
//     return;
//   }
// }
//
// static void vibrate(double duration) {
//   if (!engine) {
//     return;
//   }
//
//   @autoreleasepool {
//     NSDictionary* hapticDict = @{
//       CHHapticPatternKeyPattern: @[
//         @{
//           CHHapticPatternKeyEvent: @{
//             CHHapticPatternKeyEventType:CHHapticEventTypeHapticContinuous,
//             CHHapticPatternKeyTime:@0.0,
//             CHHapticPatternKeyEventDuration:[NSNumber numberWithDouble:duration],
//             CHHapticPatternKeyEventParameters:@[
//               @{
//                 CHHapticPatternKeyParameterID: CHHapticEventParameterIDHapticIntensity,
//                 CHHapticPatternKeyParameterValue: @1.0,
//               },
//             ],
//           },
//         },
//       ],
//     };
//
//     NSError* error = nil;
//     CHHapticPattern* pattern = [[CHHapticPattern alloc] initWithDictionary:hapticDict
//                                                                      error:&error];
//     if (error) {
//       return;
//     }
//
//     id<CHHapticPatternPlayer> player = [engine createPlayerWithPattern:pattern
//                                                                  error:&error];
//     if (error) {
//       return;
//     }
//
//     [player startAtTime:0 error:&error];
//     if (error) {
//       NSLog(@"3, %@", [error localizedDescription]);
//       return;
//     }
//   }
// }
//
import "C"

import (
	"sync"
	"time"
)

func init() {
	C.initializeVibrate()
}

var vibrationM sync.Mutex

func (u *UserInterface) Vibrate(duration time.Duration) {
	vibrationM.Lock()
	defer vibrationM.Unlock()

	C.vibrate(C.double(float64(duration) / float64(time.Second)))
}

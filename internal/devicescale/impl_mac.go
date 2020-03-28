// Copyright 2018 The Ebiten Authors
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

// +build darwin
// +build !js
// +build !ios

package devicescale

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// static float scaleAt(int x, int y) {
//   // On macOS, the direction of Y axis is inverted from GLFW monitors (#807).
//   // This is an inverse function of _glfwTransformYNS in GLFW (#1113).
//   y = CGDisplayBounds(CGMainDisplayID()).size.height - y - 1;
//
//   NSArray<NSScreen*>* screens = [NSScreen screens];
//   for (NSScreen* screen in screens) {
//     if (NSPointInRect(NSMakePoint(x, y), [screen frame])) {
//       return [screen backingScaleFactor];
//     }
//   }
//   return 0;
// }
import "C"

func impl(x, y int) float64 {
	return float64(C.scaleAt(C.int(x), C.int(y)))
}

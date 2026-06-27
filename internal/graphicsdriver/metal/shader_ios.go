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

package metal

// The iOS device and the iOS Simulator both report GOOS=ios, and there is no Go-level way to tell
// them apart. They need Metal libraries built with different SDKs, so distinguish them via the
// TARGET_OS_SIMULATOR macro, which the C compiler sets according to the SDK in use.

// #include <TargetConditionals.h>
//
// static int ebitenIsIOSSimulator(void) {
// #if TARGET_OS_SIMULATOR
//   return 1;
// #else
//   return 0;
// #endif
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/shaderprecomp"
)

func metalLibraryPlatform() shaderprecomp.MetalLibraryPlatform {
	if C.ebitenIsIOSSimulator() != 0 {
		return shaderprecomp.MetalLibraryPlatformIOSSimulator
	}
	return shaderprecomp.MetalLibraryPlatformIOS
}

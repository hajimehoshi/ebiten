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

//go:build darwin && !ios

package colormode

import (
	"bytes"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	idNSApplication = objc.ID(objc.GetClass("NSApplication"))

	selEffectiveAppearance = objc.RegisterName("effectiveAppearance")
	selLength              = objc.RegisterName("length")
	selName                = objc.RegisterName("name")
	selSharedApplication   = objc.RegisterName("sharedApplication")
	selUTF8String          = objc.RegisterName("UTF8String")
)

var (
	bytesDark = []byte("Dark")
)

func systemColorMode() ColorMode {
	// "effectiveAppearance" works from macOS 10.14. As Go 1.23 supports macOS 11, it's OK to use it.
	//
	// See also:
	// * https://developer.apple.com/documentation/appkit/nsapplication/effectiveappearance?language=objc
	// * https://go.dev/wiki/MinimumRequirements
	objcName := idNSApplication.Send(selSharedApplication).Send(selEffectiveAppearance).Send(selName)
	name := unsafe.Slice((*byte)(unsafe.Pointer(objcName.Send(selUTF8String))), objcName.Send(selLength))
	// https://developer.apple.com/documentation/appkit/nsappearance/name-swift.struct?language=objc
	if bytes.Contains(name, bytesDark) {
		return Dark
	}
	return Light
}

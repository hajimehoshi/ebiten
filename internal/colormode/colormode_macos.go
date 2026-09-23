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
	"runtime"
	"strings"

	"github.com/ebitengine/purego/cstrings"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

var (
	idNSApplication = objc.ID(objc.GetClass("NSApplication"))

	sel_effectiveAppearance = objc.RegisterName("effectiveAppearance")
	sel_name                = objc.RegisterName("name")
	sel_sharedApplication   = objc.RegisterName("sharedApplication")
)

func systemColorMode() ColorMode {
	// effectiveAppearance returns an autoreleased object. An autorelease pool is thread-local, so pin the
	// goroutine to its OS thread for the pool's lifetime; this function can be called from any goroutine.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// "effectiveAppearance" works from macOS 10.14. As Go 1.23 supports macOS 11, it's OK to use it.
	//
	// See also:
	// * https://developer.apple.com/documentation/appkit/nsapplication/effectiveappearance?language=objc
	// * https://go.dev/wiki/MinimumRequirements
	objcName := idNSApplication.Send(sel_sharedApplication).Send(sel_effectiveAppearance).Send(sel_name)
	name := cstrings.NSStringToString(objcName)
	// https://developer.apple.com/documentation/appkit/nsappearance/name-swift.struct?language=objc
	if strings.Contains(name, "Dark") {
		return Dark
	}
	return Light
}

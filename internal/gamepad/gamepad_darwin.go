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

import "runtime"

func newNativeGamepadsImpl() nativeGamepads {
	// The GameController framework handles the controllers it supports (Xbox,
	// PlayStation, MFi) correctly, including ones IOKit mis-handles.
	n := &nativeGamepadsDarwin{
		gc: newNativeGamepadsGC(),
	}

	// IOKit covers the generic HID devices the GameController framework does not
	// enumerate. It is available on macOS but not iOS, and skips the devices the
	// GameController backend claims so a shared controller is not listed twice.
	if runtime.GOOS == "darwin" {
		n.iokit = newNativeGamepadsIOKit()
	}

	return n
}

// nativeGamepadsDarwin composes the GameController and IOKit backends. gc is always
// set; iokit is nil on iOS, where IOKit is unavailable.
type nativeGamepadsDarwin struct {
	gc    nativeGamepads
	iokit nativeGamepads
}

func (g *nativeGamepadsDarwin) init(gamepads *gamepads) error {
	if err := g.gc.init(gamepads); err != nil {
		return err
	}
	if g.iokit != nil {
		if err := g.iokit.init(gamepads); err != nil {
			return err
		}
	}
	return nil
}

func (g *nativeGamepadsDarwin) update(gamepads *gamepads) error {
	if err := g.gc.update(gamepads); err != nil {
		return err
	}
	if g.iokit != nil {
		if err := g.iokit.update(gamepads); err != nil {
			return err
		}
	}
	return nil
}

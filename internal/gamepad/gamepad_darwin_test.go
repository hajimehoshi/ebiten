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

//go:build !ios

package gamepad_test

import (
	"encoding/hex"
	"fmt"
	"math/bits"
	"runtime"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

// TestGameControllerOwnsClaimedDevices checks that a device +[GCController supportsHIDDevice:]
// claims is registered by the GameController backend alone, whether or not the controller's profile
// is an extended gamepad, that the IOKit backend registers the device GameController does not claim,
// and that both hold in either order of arrival.
func TestGameControllerOwnsClaimedDevices(t *testing.T) {
	if !gamepad.SupportsTestControllers() {
		t.Skip("+[GCController controllerWithMicroGamepad] and controllerWithExtendedGamepad need macOS 11")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	if err := gamepad.InitializeCF(); err != nil {
		t.Fatal(err)
	}

	// The one device the GameController framework does not claim, so IOKit owns it.
	const hidOnlyName = "HID product name"
	const hidOnlyVendor, hidOnlyProduct = 0x045e, 0x02fd

	// Replace only the hardware boundary. CF objects, GC profiles, callbacks, registration, removal
	// and reference ownership use the production code.
	restore, err := gamepad.InstallTestHIDBoundary(func(device gamepad.HIDDeviceRef) bool {
		return gamepad.HIDDeviceProductID(device) != hidOnlyProduct
	})
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	for _, gcFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("gcFirst=%v", gcFirst), func(t *testing.T) {
			// A subtest runs on its own goroutine. The backend drains autorelease pools on the thread
			// that created them, so the goroutine must stay on one OS thread, and the virtual
			// controllers need a pool of their own on it.
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			pool := cocoa.NSAutoreleasePool_new()
			defer pool.Release()

			var pads gamepad.Gamepads
			backend := gamepad.NewDarwinGamepadsForTest()
			update := func() {
				t.Helper()
				if err := backend.Update(&pads); err != nil {
					t.Fatal(err)
				}
			}

			devices := []gamepad.HIDDeviceRef{
				gamepad.NewHIDDeviceForTest("usb gamepad", 0x0810, 0xe501),
				gamepad.NewHIDDeviceForTest("usb gamepad", 0x0810, 0xe501),
				gamepad.NewHIDDeviceForTest(hidOnlyName, hidOnlyVendor, hidOnlyProduct),
			}
			for _, d := range devices {
				defer gamepad.ReleaseHIDDeviceForTest(d)
			}
			defer pads.CloseAll()
			micro := gamepad.NewMicroControllerForTest()
			extended := gamepad.NewExtendedControllerForTest()

			connect := func() {
				gamepad.AddController(micro)
				gamepad.AddController(extended)
				update()
			}
			arrive := func() {
				for _, d := range devices {
					gamepad.HIDDeviceArrived(d)
				}
				update()
			}
			if gcFirst {
				connect()
				arrive()
			} else {
				arrive()
				connect()
			}

			var microGP, extendedGP *gamepad.Gamepad
			var gcCount int
			var hidDevices []gamepad.HIDDeviceRef
			for _, r := range pads.AppendRegisteredGamepads(nil) {
				switch r.Native {
				case gamepad.NativeGC:
					gcCount++
					if r.Micro {
						microGP = r.Gamepad
					} else {
						extendedGP = r.Gamepad
					}
				case gamepad.NativeHID:
					hidDevices = append(hidDevices, r.Device)
					if r.Name != hidOnlyName {
						t.Errorf("IOKit registered %q, want %q", r.Name, hidOnlyName)
					}
				}
			}
			if want := []gamepad.HIDDeviceRef{devices[2]}; !slices.Equal(hidDevices, want) {
				t.Errorf("IOKit registered the devices %v, want only the one GameController does not claim, %v", hidDevices, want)
			}
			if gcCount != 2 || microGP == nil || extendedGP == nil {
				t.Fatalf("registered %d GameController gamepads (micro=%v extended=%v), want one of each", gcCount, microGP != nil, extendedGP != nil)
			}

			buttonMask := microGP.GCButtonMask()
			hasMenu := buttonMask&(1<<gamepad.ControllerButtonStart) != 0

			want := uint32(1<<gamepad.ControllerButtonA | 1<<gamepad.ControllerButtonX)
			if hasMenu {
				want |= 1 << gamepad.ControllerButtonStart
			}
			if buttonMask != want {
				t.Errorf("micro GCButtonMask() = %#x, want %#x", buttonMask, want)
			}

			if got, want := microGP.AxisCount(), 2; got != want {
				t.Errorf("micro AxisCount() = %d, want %d", got, want)
			}
			if got, want := microGP.HatCount(), 1; got != want {
				t.Errorf("micro HatCount() = %d, want %d", got, want)
			}
			// buttonCount includes the four buttons of the hat.
			if got, want := microGP.ButtonCount(), bits.OnesCount32(buttonMask)+4; got != want {
				t.Errorf("micro ButtonCount() = %d, want %d", got, want)
			}
			if !microGP.IsStandardLayoutAvailable() {
				t.Error("micro IsStandardLayoutAvailable() = false, want true")
			}
			for _, b := range []gamepaddb.StandardButton{
				gamepaddb.StandardButtonRightBottom,
				gamepaddb.StandardButtonRightLeft,
				gamepaddb.StandardButtonLeftTop,
				gamepaddb.StandardButtonLeftBottom,
				gamepaddb.StandardButtonLeftLeft,
				gamepaddb.StandardButtonLeftRight,
			} {
				if !microGP.IsStandardButtonAvailable(b) {
					t.Errorf("micro IsStandardButtonAvailable(%d) = false, want true", b)
				}
			}
			for _, b := range []gamepaddb.StandardButton{
				gamepaddb.StandardButtonRightRight,
				gamepaddb.StandardButtonRightTop,
			} {
				if microGP.IsStandardButtonAvailable(b) {
					t.Errorf("micro IsStandardButtonAvailable(%d) = true, want false", b)
				}
			}
			if got := microGP.IsStandardButtonAvailable(gamepaddb.StandardButtonCenterRight); got != hasMenu {
				t.Errorf("micro IsStandardButtonAvailable(CenterRight) = %v, want %v", got, hasMenu)
			}
			// The dpad is the left stick of the standard layout.
			for _, a := range []gamepaddb.StandardAxis{
				gamepaddb.StandardAxisLeftStickHorizontal,
				gamepaddb.StandardAxisLeftStickVertical,
			} {
				if !microGP.IsStandardAxisAvailable(a) {
					t.Errorf("micro IsStandardAxisAvailable(%d) = false, want true", a)
				}
			}
			for _, a := range []gamepaddb.StandardAxis{
				gamepaddb.StandardAxisRightStickHorizontal,
				gamepaddb.StandardAxisRightStickVertical,
			} {
				if microGP.IsStandardAxisAvailable(a) {
					t.Errorf("micro IsStandardAxisAvailable(%d) = true, want false", a)
				}
			}

			guid, err := hex.DecodeString(microGP.SDLID())
			if err != nil {
				t.Fatalf("micro SDLID() = %q: %v", microGP.SDLID(), err)
			}
			if len(guid) != 16 {
				t.Fatalf("micro SDLID() decodes to %d bytes, want 16", len(guid))
			}
			for _, c := range []struct {
				index int
				want  byte
			}{
				{4, 0xac}, {5, 0x05}, // vendor 0x05ac (Apple)
				{8, 0x03}, {9, 0x00}, // product 3
				{14, 0x6d}, // 'm' for an MFi controller
				{15, 0x03}, // subtype 3
			} {
				if guid[c.index] != c.want {
					t.Errorf("micro SDLID() byte %d = %#02x, want %#02x (%s)", c.index, guid[c.index], c.want, microGP.SDLID())
				}
			}

			if err := microGP.UpdateForTest(&pads); err != nil {
				t.Fatal(err)
			}
			for i := range microGP.ButtonCountWithHats() {
				if microGP.IsButtonPressedWithHats(i) {
					t.Errorf("micro button %d pressed on an idle controller", i)
				}
			}
			for b := gamepaddb.StandardButton(0); b <= gamepaddb.StandardButtonMax; b++ {
				if microGP.IsStandardButtonPressed(b) {
					t.Errorf("micro standard button %d pressed on an idle controller", b)
				}
			}

			for _, d := range devices {
				gamepad.HIDDeviceRemoved(d)
			}
			gamepad.RemoveController(micro)
			gamepad.RemoveController(extended)
			update()
			if rs := pads.AppendRegisteredGamepads(nil); len(rs) > 0 {
				t.Errorf("%d gamepads remain after every device and controller disconnected", len(rs))
			}

			gamepad.AddController(micro)
			update()
			if !slices.ContainsFunc(pads.AppendRegisteredGamepads(nil), func(r gamepad.RegisteredGamepad) bool {
				return r.Native == gamepad.NativeGC && r.Micro
			}) {
				t.Error("micro controller missing after reconnect")
			}
		})
	}
}

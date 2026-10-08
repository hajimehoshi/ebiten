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
	"syscall/js"
	"testing"
	"time"
)

// setNavigator replaces navigator with an object whose getGamepads calls f until the test ends.
func setNavigator(t *testing.T, f func() any) {
	orig := object.Call("getOwnPropertyDescriptor", js.Global(), "navigator")
	t.Cleanup(func() {
		if orig.IsUndefined() {
			js.Global().Delete("navigator")
			return
		}
		object.Call("defineProperty", js.Global(), "navigator", orig)
	})

	getGamepads := js.FuncOf(func(this js.Value, args []js.Value) any {
		return f()
	})
	t.Cleanup(getGamepads.Release)
	nav := object.New()
	nav.Set("getGamepads", getGamepads)
	desc := object.New()
	desc.Set("value", nav)
	desc.Set("configurable", true)
	object.Call("defineProperty", js.Global(), "navigator", desc)
}

// TestUpdateWithoutGamepadsAllocs tests that update does not allocate within one second after it finds no gamepad.
func TestUpdateWithoutGamepadsAllocs(t *testing.T) {
	setNavigator(t, func() any {
		return []any{nil, nil, nil, nil}
	})

	var gps gamepads
	n := &nativeGamepadsImpl{}
	if err := n.init(&gps); err != nil {
		t.Fatal(err)
	}
	if err := n.update(&gps); err != nil {
		t.Fatal(err)
	}
	if got := testing.AllocsPerRun(100, func() {
		if err := n.update(&gps); err != nil {
			t.Fatal(err)
		}
	}); got != 0 {
		t.Errorf("allocations for each update with no gamepad: got %v, want 0", got)
	}
}

// TestUpdateFindsGamepadWithoutEvent tests that update finds a gamepad that appears without a gamepadconnected event.
func TestUpdateFindsGamepadWithoutEvent(t *testing.T) {
	gamepad := js.Null()
	setNavigator(t, func() any {
		return []any{gamepad}
	})

	var gps gamepads
	n := &nativeGamepadsImpl{}
	if err := n.init(&gps); err != nil {
		t.Fatal(err)
	}
	if err := n.update(&gps); err != nil {
		t.Fatal(err)
	}

	gamepad = object.New()
	gamepad.Set("index", 0)
	gamepad.Set("id", "test")
	gamepad.Set("mapping", "standard")
	gamepad.Set("axes", []any{})
	gamepad.Set("buttons", []any{})

	if err := n.update(&gps); err != nil {
		t.Fatal(err)
	}
	if got := len(gps.appendGamepadIDs(nil)); got != 0 {
		t.Fatalf("gamepads within one second: got %d, want 0", got)
	}

	n.lastPoll = n.lastPoll.Add(-time.Second)
	if err := n.update(&gps); err != nil {
		t.Fatal(err)
	}
	if got := len(gps.appendGamepadIDs(nil)); got != 1 {
		t.Errorf("gamepads after one second: got %d, want 1", got)
	}
	if !n.polling.Load() {
		t.Errorf("polling after a gamepad appears: got false, want true")
	}
}

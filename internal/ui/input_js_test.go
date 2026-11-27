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

package ui_test

import (
	"syscall/js"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestIsShortcutChord(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ctrl     bool
		alt      bool
		meta     bool
		altGraph bool
		isApple  bool
		want     bool
	}{
		{
			name: "no modifiers",
			want: false,
		},
		{
			name:    "no modifiers on macOS",
			isApple: true,
			want:    false,
		},
		{
			name: "Ctrl+C",
			ctrl: true,
			want: true,
		},
		{
			name:    "Ctrl+C on macOS",
			ctrl:    true,
			isApple: true,
			want:    true,
		},
		{
			name:    "Cmd+S",
			meta:    true,
			isApple: true,
			want:    true,
		},
		{
			name:    "Cmd+Option+A",
			alt:     true,
			meta:    true,
			isApple: true,
			want:    true,
		},
		{
			name: "Alt+F on Windows or Linux",
			alt:  true,
			want: true,
		},
		{
			name:    "Option+A on macOS",
			alt:     true,
			isApple: true,
			want:    false,
		},
		{
			name:     "AltGr+Q on Windows",
			ctrl:     true,
			alt:      true,
			altGraph: true,
			want:     false,
		},
		{
			name:     "AltGr+E on Linux",
			altGraph: true,
			want:     false,
		},
		{
			name:     "AltGr with Meta",
			meta:     true,
			altGraph: true,
			want:     true,
		},
		{
			name: "Ctrl+Alt without AltGraph",
			ctrl: true,
			alt:  true,
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.IsShortcutChordForTest(tt.ctrl, tt.alt, tt.meta, tt.altGraph, tt.isApple); got != tt.want {
				t.Errorf("IsShortcutChordForTest(ctrl: %t, alt: %t, meta: %t, altGraph: %t, isApple: %t): got: %t, want: %t", tt.ctrl, tt.alt, tt.meta, tt.altGraph, tt.isApple, got, tt.want)
			}
		})
	}
}

func TestIsKeyString(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want bool
	}{
		{"a", true},
		{"é", true},
		{"ḍ̇", true}, // NFC example from the UI Events key specification.
		{"क़", true},
		{"का", true}, // Spacing combining mark (Mc).
		{"ArrowLeft", false},
		{"Dead", false},
		{"a\n", false},
		{"\t", false},
		{"a\u034f", false}, // Mn with combining class zero.
		{"a\ufe0f", false}, // A variation selector has combining class zero.
	} {
		if got := ui.IsKeyStringForTest(tc.key); got != tc.want {
			t.Errorf("IsKeyString(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestPinchAfterReleasingInputs(t *testing.T) {
	for _, tc := range []struct {
		name          string
		release       bool
		newMouseEvent bool
		wantPinch     bool
	}{
		{
			name: "control_held",
		},
		{
			name:      "released",
			release:   true,
			wantPinch: true,
		},
		{
			name:          "new_control_event",
			release:       true,
			newMouseEvent: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var u ui.UserInterface
			mouseEvent := js.Global().Get("MouseEvent").New("mousemove", map[string]any{"ctrlKey": true})
			if err := u.UpdateInputFromEventForTest(mouseEvent); err != nil {
				t.Fatal(err)
			}
			if tc.release {
				u.ReleaseAllInputsForTest()
			}
			if tc.newMouseEvent {
				if err := u.UpdateInputFromEventForTest(mouseEvent); err != nil {
					t.Fatal(err)
				}
			}
			wheelEvent := js.Global().Get("WheelEvent").New("wheel", map[string]any{"ctrlKey": true, "deltaY": -10})
			if err := u.UpdateInputFromEventForTest(wheelEvent); err != nil {
				t.Fatal(err)
			}
			state := u.InputStateForTest()
			if got := state.PinchDelta != 0; got != tc.wantPinch {
				t.Errorf("pinch reported: got %t, want %t", got, tc.wantPinch)
			}
			if got := state.WheelY != 0; got == tc.wantPinch {
				t.Errorf("wheel reported: got %t, want %t", got, !tc.wantPinch)
			}
		})
	}
}

func TestKeyNameCachesLayoutAcrossTicks(t *testing.T) {
	var calls int
	name := "a"
	fail := false
	get := js.FuncOf(func(this js.Value, args []js.Value) any {
		calls++
		if fail {
			return js.Global().Get("Promise").Call("reject", "temporary failure")
		}
		m := js.Global().Get("Map").New()
		m.Call("set", "KeyA", name)
		return js.Global().Get("Promise").Call("resolve", m)
	})
	defer get.Release()
	u, cleanup := ui.NewKeyboardLayoutUIForTest(get.Value)
	defer cleanup()
	check := func(want string, wantCalls int) {
		t.Helper()
		if got := u.KeyName(ui.KeyA); got != want {
			t.Fatalf("KeyName = %q, want %q", got, want)
		}
		if calls != wantCalls {
			t.Fatalf("getLayoutMap calls = %d, want %d", calls, wantCalls)
		}
	}
	check("a", 1)
	name = "q"
	check("a", 1)
	u.ReadInputStateForTest()
	check("a", 1)
	// A layout change while focused is detected after expiry, at a tick boundary.
	u.ExpireKeyboardLayoutForTest()
	check("a", 1)
	u.ReadInputStateForTest()
	check("q", 2)
	// Transient errors remain limited to one request per tick and can recover.
	u.ExpireKeyboardLayoutForTest()
	u.ReadInputStateForTest()
	fail = true
	check("", 3)
	check("", 3)
	fail = false
	u.ReadInputStateForTest()
	check("q", 4)
}

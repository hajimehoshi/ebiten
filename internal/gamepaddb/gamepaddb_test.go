// Copyright 2022 The Ebitengine Authors
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

package gamepaddb_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func TestUpdate(t *testing.T) {
	cases := []struct {
		Input string
		Err   bool
	}{
		{
			Input: "",
			Err:   false,
		},
		{
			Input: "{}",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo",
			Err:   false,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform:Foo",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform:Windows",
			Err:   false,
		},
		{
			// An empty binding after ':' used to panic.
			Input: "00000000000000000000000000000000,foo,a:",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo,leftx:a:",
			Err:   true,
		},
	}

	for _, c := range cases {
		err := gamepaddb.Update([]byte(c.Input))
		if err == nil && c.Err {
			t.Errorf("Update(%q) should return an error but not", c.Input)
		}
		if err != nil && !c.Err {
			t.Errorf("Update(%q) should not return an error but returned %v", c.Input, err)
		}
	}
}

func TestGLFWGamepadMappings(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the current platform doesn't use GLFW gamepad mappings")
	}

	const id = "78696e70757401000000000000000000"
	if got, want := gamepaddb.HasStandardLayoutMapping(id), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := gamepaddb.Name(id), "XInput Gamepad (GLFW)"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestUpdateMappingWithoutContent(t *testing.T) {
	for _, line := range []string{
		"platform:,",
		"misc1:b5,",
	} {
		const id = "00000000000000000000000000009401"
		if err := gamepaddb.Update([]byte(id + ",Empty Pad," + line + "\n")); err != nil {
			t.Fatal(err)
		}
		if got, want := gamepaddb.HasStandardLayoutMapping(id), false; got != want {
			t.Errorf("HasStandardLayoutMapping(%q) after %q = %t; want %t", id, line, got, want)
		}
		if got, want := gamepaddb.Name(id), ""; got != want {
			t.Errorf("Name(%q) after %q = %q; want %q", id, line, got, want)
		}
	}

	const id = "00000000000000000000000000009402"
	if err := gamepaddb.Update([]byte(id + ",Test Pad,a:b0,leftx:a0,\n")); err != nil {
		t.Fatal(err)
	}
	if err := gamepaddb.Update([]byte(id + ",Test Pad,platform:,\n")); err != nil {
		t.Fatal(err)
	}
	if got, want := gamepaddb.HasStandardLayoutMapping(id), true; got != want {
		t.Errorf("HasStandardLayoutMapping(%q) = %t; want %t (a line without content must not drop a mapping)", id, got, want)
	}
	if got, want := gamepaddb.StandardButtonMapping(id, gamepaddb.StandardButtonRightBottom).IsMapped(), true; got != want {
		t.Errorf("StandardButtonMapping(%q, RightBottom).IsMapped() = %t; want %t", id, got, want)
	}
	if got, want := gamepaddb.StandardAxisMapping(id, gamepaddb.StandardAxisLeftStickHorizontal).IsMapped(), true; got != want {
		t.Errorf("StandardAxisMapping(%q, LeftStickHorizontal).IsMapped() = %t; want %t", id, got, want)
	}
}

type stubGamepadState struct {
	axes    map[int]float64
	buttons map[int]bool
}

func (s stubGamepadState) IsAxisReady(index int) bool { return true }
func (s stubGamepadState) Axis(index int) float64     { return s.axes[index] }
func (s stubGamepadState) Button(index int) bool      { return s.buttons[index] }
func (s stubGamepadState) Hat(index int) int          { return 0 }

func TestHalfAxisMappingFromDatabase(t *testing.T) {
	cases := []struct {
		GOOS           string
		Line           string
		ID             string
		PositiveButton int
		NegativeButton int
	}{
		{
			GOOS:           "windows",
			Line:           "030000007e0500001920000000000000,NSO N64 Controller,+rightx:b8,+righty:b2,-rightx:b3,-righty:b7,a:b1,b:b0,dpdown:h0.4,dpleft:h0.8,dpright:h0.2,dpup:h0.1,guide:b12,leftshoulder:b4,lefttrigger:b6,leftx:a0,lefty:a1,misc1:b13,rightshoulder:b5,righttrigger:b10,start:b9,platform:Windows,",
			ID:             "030000007e0500001920000000000000",
			PositiveButton: 8,
			NegativeButton: 3,
		},
		{
			GOOS:           "darwin",
			Line:           "030000007e0500001920000001000000,NSO N64 Controller,+rightx:b8,+righty:b7,-rightx:b3,-righty:b2,a:b1,b:b0,dpdown:h0.4,dpleft:h0.8,dpright:h0.2,dpup:h0.1,guide:b12,leftshoulder:b4,lefttrigger:b6,leftx:a0,lefty:a1,misc1:b13,rightshoulder:b5,righttrigger:b10,start:b9,platform:Mac OS X,",
			ID:             "030000007e0500001920000001000000",
			PositiveButton: 8,
			NegativeButton: 3,
		},
		{
			GOOS:           "linux",
			Line:           "050000007e0500001920000001000000,NSO N64 Controller,+rightx:b8,+righty:b7,-rightx:b3,-righty:b2,a:b1,b:b0,dpdown:h0.4,dpleft:h0.8,dpright:h0.2,dpup:h0.1,guide:b12,leftshoulder:b4,lefttrigger:b6,leftx:a0,lefty:a1,misc1:b13,rightshoulder:b5,righttrigger:b10,start:b9,platform:Linux,",
			ID:             "050000007e0500001920000001000000",
			PositiveButton: 8,
			NegativeButton: 3,
		},
	}

	tested := false
	for _, c := range cases {
		if c.GOOS != runtime.GOOS {
			continue
		}
		tested = true

		if err := gamepaddb.Update([]byte(c.Line + "\n")); err != nil {
			t.Fatal(err)
		}
		m := gamepaddb.StandardAxisMapping(c.ID, gamepaddb.StandardAxisRightStickHorizontal)
		if got, want := m.HasStandardLayout(), true; got != want {
			t.Errorf("HasStandardLayout() = %t; want %t", got, want)
		}
		if got, want := m.IsMapped(), true; got != want {
			t.Errorf("IsMapped() = %t; want %t", got, want)
		}

		released := stubGamepadState{}
		if got, want := m.AxisValue(released), 0.0; got != want {
			t.Errorf("AxisValue with no button pressed = %v; want %v", got, want)
		}
		positive := stubGamepadState{
			buttons: map[int]bool{
				c.PositiveButton: true,
			},
		}
		if got, want := m.AxisValue(positive), 1.0; got != want {
			t.Errorf("AxisValue with b%d pressed = %v; want %v", c.PositiveButton, got, want)
		}
		negative := stubGamepadState{
			buttons: map[int]bool{
				c.NegativeButton: true,
			},
		}
		if got, want := m.AxisValue(negative), -1.0; got != want {
			t.Errorf("AxisValue with b%d pressed = %v; want %v", c.NegativeButton, got, want)
		}
	}
	if !tested {
		t.Skipf("no database line for %s", runtime.GOOS)
	}
}

func TestHalfAxisMapping(t *testing.T) {
	const id = "00000000000000000000000000009403"
	cases := []struct {
		Name  string
		Line  string
		Axis  gamepaddb.StandardAxis
		State stubGamepadState
		Want  float64
	}{
		{
			Name: "positive half from a released button",
			Line: "+leftx:b0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				buttons: map[int]bool{
					0: false,
				},
			},
			Want: 0,
		},
		{
			Name: "positive half from a pressed button",
			Line: "+leftx:b0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				buttons: map[int]bool{
					0: true,
				},
			},
			Want: 1,
		},
		{
			Name: "negative half from a released button",
			Line: "-leftx:b1,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				buttons: map[int]bool{
					1: false,
				},
			},
			Want: 0,
		},
		{
			Name: "negative half from a pressed button",
			Line: "-leftx:b1,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				buttons: map[int]bool{
					1: true,
				},
			},
			Want: -1,
		},
		{
			Name: "both halves from two buttons, none pressed",
			Line: "+righty:b2,-righty:b3,",
			Axis: gamepaddb.StandardAxisRightStickVertical,
			State: stubGamepadState{
				buttons: map[int]bool{
					2: false,
					3: false,
				},
			},
			Want: 0,
		},
		{
			Name: "both halves from two buttons, positive pressed",
			Line: "+righty:b2,-righty:b3,",
			Axis: gamepaddb.StandardAxisRightStickVertical,
			State: stubGamepadState{
				buttons: map[int]bool{
					2: true,
					3: false,
				},
			},
			Want: 1,
		},
		{
			Name: "both halves from two buttons, negative pressed",
			Line: "+righty:b2,-righty:b3,",
			Axis: gamepaddb.StandardAxisRightStickVertical,
			State: stubGamepadState{
				buttons: map[int]bool{
					2: false,
					3: true,
				},
			},
			Want: -1,
		},
		{
			Name: "both halves from two buttons, both pressed",
			Line: "+righty:b2,-righty:b3,",
			Axis: gamepaddb.StandardAxisRightStickVertical,
			State: stubGamepadState{
				buttons: map[int]bool{
					2: true,
					3: true,
				},
			},
			Want: 0,
		},
		{
			Name: "positive half from a half-axis input at rest",
			Line: "+leftx:+a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: 0,
				},
			},
			Want: 0,
		},
		{
			Name: "positive half from a half-axis input, half way",
			Line: "+leftx:+a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: 0.5,
				},
			},
			Want: 0.5,
		},
		{
			Name: "positive half from a half-axis input, full",
			Line: "+leftx:+a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: 1,
				},
			},
			Want: 1,
		},
		{
			Name: "positive half from a half-axis input, the other half is ignored",
			Line: "+leftx:+a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: -1,
				},
			},
			Want: 0,
		},
		{
			Name: "negative half from a negative half-axis input",
			Line: "-leftx:-a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: -0.5,
				},
			},
			Want: -0.5,
		},
		{
			Name: "both halves from two half-axis inputs, negative",
			Line: "+lefty:+a2,-lefty:-a1,",
			Axis: gamepaddb.StandardAxisLeftStickVertical,
			State: stubGamepadState{
				axes: map[int]float64{
					1: -0.25,
					2: 0,
				},
			},
			Want: -0.25,
		},
		{
			Name: "both halves from two half-axis inputs, positive",
			Line: "+lefty:+a2,-lefty:-a1,",
			Axis: gamepaddb.StandardAxisLeftStickVertical,
			State: stubGamepadState{
				axes: map[int]float64{
					1: 0,
					2: 0.75,
				},
			},
			Want: 0.75,
		},
		{
			Name: "positive half from a whole-axis input",
			Line: "+rightx:a3,",
			Axis: gamepaddb.StandardAxisRightStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					3: 0,
				},
			},
			Want: 0.5,
		},
		{
			Name: "whole axis from a whole-axis input",
			Line: "leftx:a0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				axes: map[int]float64{
					0: -0.5,
				},
			},
			Want: -0.5,
		},
		{
			Name: "whole axis from a button",
			Line: "leftx:b0,",
			Axis: gamepaddb.StandardAxisLeftStickHorizontal,
			State: stubGamepadState{
				buttons: map[int]bool{
					0: false,
				},
			},
			Want: -1,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if err := gamepaddb.Update([]byte(id + ",Test Pad," + c.Line + "\n")); err != nil {
				t.Fatal(err)
			}
			m := gamepaddb.StandardAxisMapping(id, c.Axis)
			if got, want := m.IsMapped(), true; got != want {
				t.Errorf("IsMapped() = %t; want %t", got, want)
			}
			if got, want := m.AxisValue(c.State), c.Want; got != want {
				t.Errorf("AxisValue() = %v; want %v", got, want)
			}
		})
	}
}

// mockGamepad mimics internal/gamepad.Gamepad:
// every state query takes the gamepad's own mutex.
type mockGamepad struct {
	mu sync.Mutex
}

func (g *mockGamepad) IsAxisReady(index int) bool { g.mu.Lock(); defer g.mu.Unlock(); return true }
func (g *mockGamepad) Axis(index int) float64     { g.mu.Lock(); defer g.mu.Unlock(); return 0 }
func (g *mockGamepad) Button(index int) bool      { g.mu.Lock(); defer g.mu.Unlock(); return false }
func (g *mockGamepad) Hat(index int) int          { g.mu.Lock(); defer g.mu.Unlock(); return 0 }

// queryWithGamepadLock calls this package while the gamepad's own lock is held.
func queryWithGamepadLock(g *mockGamepad, id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	_ = gamepaddb.HasStandardLayoutMapping(id)
	_ = gamepaddb.StandardAxisMapping(id, gamepaddb.StandardAxisLeftStickHorizontal)
	_ = gamepaddb.StandardButtonMapping(id, gamepaddb.StandardButtonRightBottom)
	_ = gamepaddb.Name(id)
}

// TestConcurrentAccess uses this package from multiple goroutines in both lock orders:
// one takes the gamepad's own lock before calling in, and the other lets this package
// call back into the gamepad. Holding mappingsM while a gamepad state is used makes the
// two orders wait for each other.
//
// This must be the last test in this file, as a deadlock leaves mappingsM held
// and any test running after it would hang.
func TestConcurrentAccess(t *testing.T) {
	const id = "00000000000000000000000000000001"
	mappings := []byte(id + ",Test Pad,a:b0,leftx:a0,\n")
	if err := gamepaddb.Update(mappings); err != nil {
		t.Fatal(err)
	}

	g := &mockGamepad{}
	deadline := time.Now().Add(200 * time.Millisecond)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for time.Now().Before(deadline) {
				queryWithGamepadLock(g, id)
			}
		})

		wg.Go(func() {
			for time.Now().Before(deadline) {
				_ = gamepaddb.StandardAxisMapping(id, gamepaddb.StandardAxisLeftStickHorizontal).AxisValue(g)
				_ = gamepaddb.StandardButtonMapping(id, gamepaddb.StandardButtonRightBottom).ButtonValue(g)
				_ = gamepaddb.StandardButtonMapping(id, gamepaddb.StandardButtonRightBottom).IsButtonPressed(g)
			}
		})
	}

	wg.Go(func() {
		for time.Now().Before(deadline) {
			_ = gamepaddb.Update(mappings)
		}
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlocked: a gamepad state was probably used while mappingsM was held")
	}
}

func TestAddAndroidDefaultMappings(t *testing.T) {
	// A well-formed Android gamepad ID consists of 16 bytes i.e. 32 hex characters.
	const validID = "0000000000000000000000000f000000"

	if got, want := gamepaddb.AddAndroidDefaultMappings(validID), true; got != want {
		t.Errorf("gamepaddb.AddAndroidDefaultMappings(%q): got: %t, want: %t", validID, got, want)
	}

	// A shorter ID must be rejected instead of causing a panic.
	for n := range len(validID) {
		id := validID[:n]
		if got, want := gamepaddb.AddAndroidDefaultMappings(id), false; got != want {
			t.Errorf("gamepaddb.AddAndroidDefaultMappings(%q): got: %t, want: %t", id, got, want)
		}
	}
}

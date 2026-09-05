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

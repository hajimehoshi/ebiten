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

// gamepadLike mimics internal/gamepad.Gamepad:
// every state query takes the gamepad's own mutex.
type gamepadLike struct {
	mu sync.Mutex
}

func (g *gamepadLike) IsAxisReady(index int) bool { g.mu.Lock(); defer g.mu.Unlock(); return true }
func (g *gamepadLike) Axis(index int) float64     { g.mu.Lock(); defer g.mu.Unlock(); return 0 }
func (g *gamepadLike) Button(index int) bool      { g.mu.Lock(); defer g.mu.Unlock(); return false }
func (g *gamepadLike) Hat(index int) int          { g.mu.Lock(); defer g.mu.Unlock(); return 0 }

// queryWithGamepadLock calls this package while the gamepad's own lock is held,
// like Gamepad.IsStandardAxisAvailable does.
func queryWithGamepadLock(g *gamepadLike, id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	_ = gamepaddb.HasStandardLayoutMapping(id)
	_ = gamepaddb.HasStandardAxis(id, gamepaddb.StandardAxisLeftStickHorizontal)
	_ = gamepaddb.HasStandardButton(id, gamepaddb.StandardButtonRightBottom)
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

	g := &gamepadLike{}
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
				_ = gamepaddb.StandardAxisValue(id, gamepaddb.StandardAxisLeftStickHorizontal, g)
				_ = gamepaddb.StandardButtonValue(id, gamepaddb.StandardButtonRightBottom, g)
				_ = gamepaddb.IsStandardButtonPressed(id, gamepaddb.StandardButtonRightBottom, g)
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

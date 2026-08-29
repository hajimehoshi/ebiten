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
	"fmt"
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

func TestAndroidDefaultMappingsConcurrent(t *testing.T) {
	old := gamepaddb.SetPlatformForTesting(gamepaddb.PlatformAndroidForTesting)
	defer gamepaddb.SetPlatformForTesting(old)

	const goroutines = 16
	ids := make([]string, goroutines)
	for g := range goroutines {
		ids[g] = fmt.Sprintf("030000004c0500006802000001%02x0000", g)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			for range 1000 {
				_ = gamepaddb.HasStandardButton(ids[g], gamepaddb.StandardButtonRightBottom)
			}
		}(g)
	}
	close(start)
	wg.Wait()

	for g := range goroutines {
		if !gamepaddb.HasStandardButton(ids[g], gamepaddb.StandardButtonRightBottom) {
			t.Errorf("the Android default mapping was not registered for %s", ids[g])
		}
	}
}

// gamepadLike mimics internal/gamepad.Gamepad:
// every state query takes the gamepad's own mutex.
type gamepadLike struct {
	mu sync.Mutex
	// entered is closed when a state method is called,
	// and the method waits for proceed to be closed before taking mu.
	entered chan struct{}
	proceed chan struct{}
	once    sync.Once
}

// handOver lets the goroutine which holds the gamepad lock and calls into gamepaddb run.
func (g *gamepadLike) handOver() {
	g.once.Do(func() {
		close(g.entered)
		<-g.proceed
	})
}

func (g *gamepadLike) IsAxisReady(index int) bool {
	g.handOver()
	g.mu.Lock()
	defer g.mu.Unlock()
	return true
}

func (g *gamepadLike) Axis(index int) float64 {
	g.handOver()
	g.mu.Lock()
	defer g.mu.Unlock()
	return 0
}

func (g *gamepadLike) Button(index int) bool {
	g.handOver()
	g.mu.Lock()
	defer g.mu.Unlock()
	return false
}

func (g *gamepadLike) Hat(index int) int {
	g.handOver()
	g.mu.Lock()
	defer g.mu.Unlock()
	return 0
}

// TestLockOrderInversion checks that gamepaddb does not hold its lock
// while calling a gamepad state, which would deadlock with the other order,
// where a gamepad holds its own lock and then calls into gamepaddb.
// It detects the deadlock even without -race, which CI does not run.
func TestLockOrderInversion(t *testing.T) {
	const id = "00000000000000000000000000000001"
	if err := gamepaddb.Update([]byte(id + ",Test Pad,a:b0,leftx:a0,\n")); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		f    func(state gamepaddb.GamepadState)
	}{
		{
			name: "StandardAxisValue",
			f: func(state gamepaddb.GamepadState) {
				gamepaddb.StandardAxisValue(id, gamepaddb.StandardAxisLeftStickHorizontal, state)
			},
		},
		{
			name: "StandardButtonValue",
			f: func(state gamepaddb.GamepadState) {
				gamepaddb.StandardButtonValue(id, gamepaddb.StandardButtonRightBottom, state)
			},
		},
		{
			name: "IsStandardButtonPressed",
			f: func(state gamepaddb.GamepadState) {
				gamepaddb.IsStandardButtonPressed(id, gamepaddb.StandardButtonRightBottom, state)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &gamepadLike{
				entered: make(chan struct{}),
				proceed: make(chan struct{}),
			}
			done := make(chan struct{})

			go func() {
				defer close(done)
				tc.f(g)
			}()

			mustReceive(t, g.entered, "the gamepad state was not used; gamepaddb is probably blocked on its own lock by a previous deadlock")

			go func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				close(g.proceed)
				gamepaddb.HasStandardLayoutMapping(id)
				gamepaddb.HasStandardAxis(id, gamepaddb.StandardAxisLeftStickHorizontal)
				gamepaddb.HasStandardButton(id, gamepaddb.StandardButtonRightBottom)
				gamepaddb.Name(id)
			}()

			mustReceive(t, done, "deadlocked: mappingsM must not be held while a gamepad state is used")
		})
	}
}

func mustReceive(t *testing.T, c chan struct{}, msg string) {
	t.Helper()

	select {
	case <-c:
	case <-time.After(10 * time.Second):
		t.Fatal(msg)
	}
}

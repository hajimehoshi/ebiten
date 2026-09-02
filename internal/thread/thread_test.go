// Copyright 2026 The Ebiten Authors
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

package thread_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func TestNestedLoop(t *testing.T) {
	th := thread.NewOSThread()

	ctx := t.Context()
	go func() {
		_ = th.Loop(ctx)
	}()

	var values []int
	th.Call(func() {
		values = append(values, 1)

		// While this function blocks the thread, another goroutine's Call must be
		// processed by NestedLoop.
		nestedCtx, nestedCancel := context.WithCancel(context.Background())
		go func() {
			defer nestedCancel()
			th.Call(func() {
				values = append(values, 2)
			})
		}()
		_ = th.NestedLoop(nestedCtx)

		values = append(values, 3)
	})
	th.Call(func() {
		values = append(values, 4)
	})

	if got, want := values, []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestCallAfterLoopAndStop(t *testing.T) {
	th := thread.NewOSThread()

	ctx, cancel := context.WithCancel(context.Background())
	loopEnded := make(chan struct{})
	go func() {
		defer close(loopEnded)
		_ = th.LoopAndStop(ctx)
	}()

	// Make sure that the loop is running.
	th.Call(func() {})

	cancel()
	<-loopEnded

	var called bool
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		th.Call(func() {
			called = true
		})
		th.CallAsync(func() {
			called = true
		})
	}()

	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("Call after LoopAndStop must return")
	}

	if called {
		t.Error("the function must not be called after LoopAndStop")
	}
}

func TestLoopAndStopUnblocksCall(t *testing.T) {
	th := thread.NewOSThread()

	// No loop is running, so this Call blocks until the thread stops.
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		th.Call(func() {})
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = th.LoopAndStop(ctx)

	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("LoopAndStop must unblock a blocking Call")
	}
}

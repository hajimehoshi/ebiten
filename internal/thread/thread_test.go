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

package thread_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
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
	thread.Call(th, func() {
		values = append(values, 1)

		// While this function blocks the thread, another goroutine's Call must be
		// processed by NestedLoop.
		nestedCtx, nestedCancel := context.WithCancel(context.Background())
		go func() {
			defer nestedCancel()
			thread.Call(th, func() {
				values = append(values, 2)
			})
		}()
		_ = th.NestedLoop(nestedCtx)

		values = append(values, 3)
	})
	thread.Call(th, func() {
		values = append(values, 4)
	})

	if got, want := values, []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestLoopAndStopUnblocksCall(t *testing.T) {
	th := thread.NewOSThread()

	// No loop is running, so this Call blocks until the thread stops.
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		thread.Call(th, func() {})
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

func TestCalls(t *testing.T) {
	for _, name := range []string{"NoopThread", "OSThread"} {
		t.Run(name, func(t *testing.T) {
			var th thread.Thread = thread.NewNoopThread()
			if name == "OSThread" {
				th = thread.NewOSThread()
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = th.Loop(ctx)
			}()
			defer func() { cancel(); <-done }()

			type argument struct {
				value  *int
				amount int
			}
			add := func(a argument) int {
				*a.value += a.amount
				return *a.value
			}
			var value int
			if got := thread.CallWithArgAndResult(th, add, argument{value: &value, amount: 3}); got != 3 {
				t.Errorf("Call result = %d, want 3", got)
			}
			thread.CallAsync(th, func(a argument) { *a.value += a.amount }, argument{value: &value, amount: 4})
			// The synchronous call observes the preceding asynchronous call's result.
			if got := thread.CallWithArgAndResult(th, add, argument{value: &value, amount: 5}); got != 12 {
				t.Errorf("Call result after CallAsync = %d, want 12", got)
			}
			thread.CallWithArg(th, func(a argument) { *a.value += a.amount }, argument{value: &value, amount: 2})
			if value != 14 {
				t.Errorf("value after CallWithArg = %d, want 14", value)
			}
			var called bool
			thread.Call(th, func() { called = true })
			if !called {
				t.Error("Call did not execute its callback")
			}
			for _, want := range []error{context.Canceled, nil} {
				if got := thread.CallWithArgAndResult(th, func(err error) error { return err }, want); got != want {
					t.Errorf("Call error result = %v, want %v", got, want)
				}
			}
			cancel()
			<-done
			if err := th.LoopAndStop(ctx); err != nil && err != context.Canceled {
				t.Fatal(err)
			}
			if got := thread.CallWithArgAndResult(th, add, argument{value: &value, amount: 6}); got != 0 {
				t.Errorf("stopped Call result = %d, want 0", got)
			}
			thread.CallAsync(th, func(a argument) { *a.value += a.amount }, argument{value: &value, amount: 7})
			thread.CallWithArg(th, func(a argument) { *a.value += a.amount }, argument{value: &value, amount: 8})
			thread.Call(th, func() { called = false })
			if !called {
				t.Error("Call executed its callback after stopping")
			}
			if value != 14 {
				t.Errorf("value after stopped calls = %d, want 14", value)
			}
		})
	}
}

func TestTypedNestedCall(t *testing.T) {
	th := thread.NewOSThread()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = th.LoopAndStop(ctx)
	}()
	defer func() { cancel(); <-done }()

	got := thread.CallWithArgAndResult(th, func(value int) int {
		nestedCtx, nestedCancel := context.WithCancel(context.Background())
		nestedResult := make(chan int, 1)
		go func() {
			defer nestedCancel()
			nestedResult <- thread.CallWithArgAndResult(th, func(value int) int { return value * 2 }, value)
		}()
		_ = th.NestedLoop(nestedCtx)
		return <-nestedResult + 1
	}, 20)
	if got != 41 {
		t.Errorf("nested Call result = %d, want 41", got)
	}
}

func TestConcurrentTypedCalls(t *testing.T) {
	th := thread.NewOSThread()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = th.LoopAndStop(ctx)
	}()
	defer func() { cancel(); <-done }()

	const callers = 16
	const calls = 32
	var completed atomic.Int32
	f := func(value int) int {
		completed.Add(1)
		return value * 2
	}
	var wg sync.WaitGroup
	for caller := range callers {
		wg.Go(func() {
			for i := range calls {
				value := caller*calls + i
				thread.CallAsync(th, func(value int) { completed.Add(1) }, value)
				if got := thread.CallWithArgAndResult(th, f, value); got != value*2 {
					t.Errorf("Call(%d) = %d, want %d", value, got, value*2)
				}
				var got int
				thread.CallWithArg(th, func(value int) { got = value }, value)
				if got != value {
					t.Errorf("CallWithArg(%d) wrote %d", value, got)
				}
				thread.Call(th, func() { got++ })
				if got != value+1 {
					t.Errorf("Call wrote %d, want %d", got, value+1)
				}
			}
		})
	}
	wg.Wait()
	if got := completed.Load(); got != 2*callers*calls {
		t.Errorf("completed calls = %d, want %d", got, 2*callers*calls)
	}
}

func BenchmarkCall(b *testing.B) {
	for _, name := range []string{"NoopThread", "OSThread"} {
		b.Run(name, func(b *testing.B) {
			var th thread.Thread = thread.NewNoopThread()
			if name == "OSThread" {
				th = thread.NewOSThread()
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan struct{})
				go func() {
					defer close(done)
					_ = th.LoopAndStop(ctx)
				}()
				defer func() { cancel(); <-done }()
			}
			b.Run("NoArg", func(b *testing.B) {
				f := func() {}
				thread.Call(th, f)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					thread.Call(th, f)
				}
			})
			b.Run("WithArgAndResult", func(b *testing.B) {
				f := func(v int) int { return v + 1 }
				thread.CallWithArgAndResult(th, f, 1)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					thread.CallWithArgAndResult(th, f, i)
				}
			})
			b.Run("WithArg", func(b *testing.B) {
				f := func(_ int) {}
				thread.CallWithArg(th, f, 1)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					thread.CallWithArg(th, f, i)
				}
			})
			b.Run("Async", func(b *testing.B) {
				f := func(_ int) {}
				thread.CallAsync(th, f, 1)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					thread.CallAsync(th, f, i)
				}
				b.StopTimer()
				thread.Call(th, func() {})
			})
		})
	}
}

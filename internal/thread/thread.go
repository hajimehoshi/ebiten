// Copyright 2018 The Ebiten Authors
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

package thread

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
)

type Thread interface {
	Loop(ctx context.Context) error
	LoopAndStop(ctx context.Context) error

	call(f callable, sync bool) bool
}

type callable interface {
	call()
}

var callPools sync.Map

type syncCall[A, R any] struct {
	f      func(A) R
	arg    A
	result R
}

func (c *syncCall[A, R]) call() {
	// The waiting caller needs c.result intact and releases c after copying the result.
	c.result = c.f(c.arg)
}

type voidCall[A any] struct {
	f   func(A)
	arg A
}

func (c *voidCall[A]) call() {
	c.f(c.arg)
}

type asyncCall[A any] struct {
	pool *sync.Pool
	f    func(A)
	arg  A
}

func (c *asyncCall[A]) call() {
	defer releaseCall(c, c.pool)
	c.f(c.arg)
}

func getCall[C any]() (*C, *sync.Pool) {
	key := reflect.TypeFor[C]()
	p, ok := callPools.Load(key)
	if !ok {
		p, _ = callPools.LoadOrStore(key, &sync.Pool{
			New: func() any { return new(C) },
		})
	}
	pool := p.(*sync.Pool)
	return pool.Get().(*C), pool
}

func releaseCall[C any](c *C, pool *sync.Pool) {
	var zero C
	*c = zero
	pool.Put(c)
}

// Call calls f on t and waits for completion, or does nothing if t has stopped.
// On an OSThread, Call blocks until Loop executes f and must not be called from that thread.
// On a NoopThread, Call executes f immediately.
func Call(t Thread, f func()) {
	CallWithArg(t, func(f func()) { f() }, f)
}

// CallWithArgAndResult calls f with arg on t and returns its result, or the zero value if t has stopped.
// The same blocking restrictions as Call apply.
func CallWithArgAndResult[A, R any](t Thread, f func(A) R, arg A) R {
	if t, ok := t.(*NoopThread); ok {
		if t.stopped.Load() {
			logDiscardedCall()
			var zero R
			return zero
		}
		return f(arg)
	}
	c, pool := getCall[syncCall[A, R]]()
	defer releaseCall(c, pool)
	c.f = f
	c.arg = arg
	t.call(c, true)
	return c.result
}

// CallWithArg calls f with arg on t and waits for completion, or does nothing if t has stopped.
// The same blocking restrictions as Call apply.
func CallWithArg[A any](t Thread, f func(A), arg A) {
	if t, ok := t.(*NoopThread); ok {
		if t.stopped.Load() {
			logDiscardedCall()
			return
		}
		f(arg)
		return
	}
	c, pool := getCall[voidCall[A]]()
	defer releaseCall(c, pool)
	c.f = f
	c.arg = arg
	t.call(c, true)
}

// CallAsync queues f with arg on t.
// On an OSThread, CallAsync blocks until Loop accepts f and must not be called from that thread.
// On a NoopThread, CallAsync executes f immediately. It does nothing if t has stopped.
func CallAsync[A any](t Thread, f func(A), arg A) {
	if t, ok := t.(*NoopThread); ok {
		if t.stopped.Load() {
			logDiscardedCall()
			return
		}
		f(arg)
		return
	}
	c, pool := getCall[asyncCall[A]]()
	c.pool = pool
	c.f = f
	c.arg = arg
	// A successful enqueue transfers ownership to the worker, which releases c after execution.
	// The worker may still be using c when t.call returns, so only a discarded call can be released here.
	if !t.call(c, false) {
		releaseCall(c, pool)
	}
}

type queueItem struct {
	f callable

	// done is closed when the execution of f is completed. done is nil for an asynchronous call.
	//
	// done must be dedicated to one queue item. With NestedLoop, multiple synchronous calls can
	// be in flight at the same time, and a completion signal on a shared channel could be
	// received by a wrong caller.
	done chan struct{}
}

// OSThread represents an OS thread.
type OSThread struct {
	funcs chan queueItem

	// stopped is closed at stop.
	stopped  chan struct{}
	stopOnce sync.Once
}

// NewOSThread creates a new thread.
func NewOSThread() *OSThread {
	return &OSThread{
		funcs:   make(chan queueItem),
		stopped: make(chan struct{}),
	}
}

// Loop starts the thread loop until ctx is canceled.
//
// Loop returns ctx's error if exists.
//
// Loop can be called again after it returns. Use LoopAndStop for a loop that is the
// thread's whole life.
//
// Loop must be called on the OS thread.
func (t *OSThread) Loop(ctx context.Context) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return t.loop(ctx)
}

// LoopAndStop runs Loop and then stops the thread.
//
// After LoopAndStop returns, Call and CallAsync do nothing.
//
// LoopAndStop returns ctx's error if exists.
//
// LoopAndStop must be called on the OS thread.
func (t *OSThread) LoopAndStop(ctx context.Context) error {
	defer t.stop()
	return t.Loop(ctx)
}

// NestedLoop runs functions requested by Call and CallAsync until ctx is canceled.
//
// NestedLoop is useful when a function invoked by Loop blocks the thread and requests
// from other goroutines must be processed in the meantime.
//
// NestedLoop returns ctx's error if exists.
//
// NestedLoop must be called on the OS thread running Loop, from within a function invoked by Loop.
func (t *OSThread) NestedLoop(ctx context.Context) error {
	return t.loop(ctx)
}

func (t *OSThread) loop(ctx context.Context) error {
	for {
		select {
		case item := <-t.funcs:
			func() {
				if item.done != nil {
					defer close(item.done)
				}
				item.f.call()
			}()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// stop stops serving Call and CallAsync.
//
// stop can be called multiple times.
func (t *OSThread) stop() {
	t.stopOnce.Do(func() {
		close(t.stopped)
	})
}

func (t *OSThread) call(f callable, sync bool) bool {
	var done chan struct{}
	if sync {
		done = make(chan struct{})
	}
	select {
	case t.funcs <- queueItem{f: f, done: done}:
	case <-t.stopped:
		logDiscardedCall()
		return false
	}
	if sync {
		<-done
	}
	return true
}

// NoopThread is used to disable threading.
type NoopThread struct {
	stopped atomic.Bool
}

// NewNoopThread creates a new thread that does no threading.
func NewNoopThread() *NoopThread {
	return &NoopThread{}
}

// Loop does nothing.
func (t *NoopThread) Loop(ctx context.Context) error {
	return nil
}

// LoopAndStop stops the thread without looping.
//
// After LoopAndStop returns, Call and CallAsync do nothing.
func (t *NoopThread) LoopAndStop(ctx context.Context) error {
	t.stop()
	return nil
}

// stop stops serving Call and CallAsync.
//
// stop can be called multiple times.
func (t *NoopThread) stop() {
	t.stopped.Store(true)
}

func (t *NoopThread) call(f callable, sync bool) bool {
	if t.stopped.Load() {
		logDiscardedCall()
		return false
	}
	f.call()
	return true
}

// logDiscardedCall records a call that was not executed as the thread was stopped.
func logDiscardedCall() {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	var caller string
	file, line, typ := debug.FirstCaller()
	switch typ {
	case debug.CallerTypeRegular:
		caller = fmt.Sprintf("%s:%d", file, line)
	case debug.CallerTypeInternal:
		caller = fmt.Sprintf("%s:%d (internal)", file, line)
	}
	slog.Debug("thread: a call was discarded as the thread was stopped", "caller", caller)
}

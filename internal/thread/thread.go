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

import "sync"

// Thread defines threading behavior in Ebiten.
type Thread interface {
	Call(func())
	Loop()
	Stop()
}

type funcdata struct {
	f    func()
	done chan struct{}
}

// OSThread represents an OS thread.
type OSThread struct {
	funcs chan funcdata
	done  chan struct{}
}

// NewOSThread creates a new thread.
//
// It is assumed that the OS thread is fixed by runtime.LockOSThread when NewOSThread is called.
func NewOSThread() *OSThread {
	return &OSThread{
		funcs: make(chan funcdata),
		done:  make(chan struct{}),
	}
}

// Loop starts the thread loop until Stop is called.
//
// Loop must be called on the thread.
func (t *OSThread) Loop() {
	for {
		select {
		case fn := <-t.funcs:
			fn.f()
			if fn.done != nil {
				close(fn.done)
			}
		case <-t.done:
			return
		}
	}
}

// Stop stops the thread loop.
func (t *OSThread) Stop() {
	close(t.done)
}

var donePool = sync.Pool{
	New: func() interface{} {
		return make(chan struct{})
	},
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call blocks if Loop is not called.
func (t *OSThread) Call(f func()) {
	done := donePool.Get().(chan struct{})
	defer donePool.Put(done)

	t.funcs <- funcdata{
		f:    f,
		done: done,
	}
	<-done
}

// NoopThread is used to disable threading.
type NoopThread struct{}

// NewNoopThread creates a new thread that does no threading.
func NewNoopThread() *NoopThread {
	return &NoopThread{}
}

// Loop does nothing
func (t *NoopThread) Loop() {}

// Call executes the func immediately
func (t *NoopThread) Call(f func() error) error {
	return f()
}

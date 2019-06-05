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
	"sync/atomic"
)

// Thread represents an OS thread.
type Thread struct {
	started int32
	funcs   chan func()
}

// New creates a new thread.
//
// It is assumed that the OS thread is fixed by runtime.LockOSThread when New is called.
func New() *Thread {
	return &Thread{
		funcs: make(chan func()),
	}
}

// Loop starts the thread loop.
//
// Loop must be called on the thread.
func (t *Thread) Loop(ch <-chan error) error {
	atomic.StoreInt32(&t.started, 1)
	for {
		select {
		case f := <-t.funcs:
			f()
		case err := <-ch:
			// ch returns a value not only when an error occur but also it is closed.
			return err
		}
	}
}

// Run calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
func (t *Thread) Run(f func() error) error {
	if atomic.LoadInt32(&t.started) == 0 {
		panic("thread: the thread loop is not started yet")
	}

	ch := make(chan struct{})
	var err error
	t.funcs <- func() {
		err = f()
		close(ch)
	}
	<-ch
	return err
}

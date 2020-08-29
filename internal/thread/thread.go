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
	"errors"
)

// Thread represents an OS thread.
type Thread struct {
	funcs   chan func() error
	results chan error
}

// New creates a new thread.
//
// It is assumed that the OS thread is fixed by runtime.LockOSThread when New is called.
func New() *Thread {
	return &Thread{
		funcs:   make(chan func() error),
		results: make(chan error),
	}
}

// BreakLoop represents an termination of the loop.
var BreakLoop = errors.New("break loop")

// Loop starts the thread loop until a posted function returns BreakLoop.
//
// Loop must be called on the thread.
func (t *Thread) Loop() {
	for f := range t.funcs {
		err := f()
		if err == BreakLoop {
			t.results <- nil
			return
		}
		t.results <- err
	}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// If f returns BreakLoop, Loop returns.
//
// Call blocks if Loop is not called.
func (t *Thread) Call(f func() error) error {
	t.funcs <- f
	return <-t.results
}

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

package mainthread

import (
	"runtime"
	"sync"
)

var (
	funcs   chan func()
	running bool

	m sync.Mutex
)

func init() {
	runtime.LockOSThread()
}

// Loop starts the main-thread loop.
//
// Loop must be called on the main thread.
func Loop(ch <-chan error) error {
	m.Lock()
	funcs = make(chan func())
	m.Unlock()
	for {
		select {
		case f := <-funcs:
			m.Lock()
			running = true
			m.Unlock()
			f()
			m.Lock()
			running = false
			m.Unlock()
		case err := <-ch:
			// ch returns a value not only when an error occur but also it is closed.
			return err
		}
	}
}

// Run calls f on the main thread.
//
// Run can be called even before Loop is called.
//
// Run can be called recursively: Run can be called from the function that are called via Run.
func Run(f func() error) error {
	// Even if funcs is nil, Run is called from the main thread (e.g. init)
	m.Lock()
	now := funcs == nil || running
	m.Unlock()
	if now {
		return f()
	}

	ch := make(chan struct{})
	var err error
	funcs <- func() {
		err = f()
		close(ch)
	}
	<-ch
	return err
}

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
	"sync/atomic"
)

func init() {
	runtime.LockOSThread()
}

var (
	started = int32(0)
	funcs   = make(chan func())
)

// Loop starts the main-thread loop.
//
// Loop must be called on the main thread.
func Loop(ch <-chan error) error {
	atomic.StoreInt32(&started, 1)
	for {
		select {
		case f := <-funcs:
			f()
		case err := <-ch:
			// ch returns a value not only when an error occur but also it is closed.
			return err
		}
	}
}

// Run calls f on the main thread.
//
// Do not call this from the main thread. This would block forever.
func Run(f func() error) error {
	if atomic.LoadInt32(&started) == 0 {
		// TODO: This can reach from other goroutine before Loop is called (#809).
		// panic("mainthread: the mainthread loop is not started yet")
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

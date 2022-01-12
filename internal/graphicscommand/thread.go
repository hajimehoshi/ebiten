// Copyright 2020 The Ebiten Authors
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

package graphicscommand

import (
	"runtime"
	"sync"
)

var (
	theThread    Thread
	theThreadId  uint64
	mainthreadId uint64

	mu      sync.Mutex
	pending []func()
)

func init() {
	mainthreadId = gid()
}

func gid() (id uint64) {
	var buf [30]byte
	runtime.Stack(buf[:], false)
	for i := 10; buf[i] != ' '; i++ {
		id = id*10 + uint64(buf[i]&15)
	}
	return id
}

type Thread interface {
	Call(f func())
}

// SetMainThread must be called from the main thread (i.e, the goroutine where the thread is created).
func SetMainThread(thread Thread) {
	theThread = thread
	theThreadId = gid()
}

// RunOnMainThread calls f on the main thread, and returns an error if any.
func RunOnMainThread(f func()) {
	if theThreadId != mainthreadId {
		mu.Lock()
		pending = append(pending, f)
		mu.Unlock()
		return
	}

	// The thread is nil when 1) GOOS=js or 2) using golang.org/x/mobile/gl.
	// When golang.org/x/mobile/gl is used, all the GL functions are called via Context, which already runs on an
	// appropriate thread.
	var caller func(func())
	if theThread == nil {
		caller = func(f func()) { f() }
	} else {
		caller = theThread.Call
	}

	mu.Lock()
	if len(pending) > 0 {
		for i := range pending {
			caller(pending[i])
		}
		pending = []func(){}
	}
	mu.Unlock()
	caller(f)
}

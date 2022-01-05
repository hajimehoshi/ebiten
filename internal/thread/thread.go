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

// Thread defines threading behavior in Ebiten.
type Thread interface {
	Call(func())
	Loop()
	Stop()
}

// OSThread represents an OS thread.
type OSThread struct {
	funcs     chan func()
	done      chan struct{}
	terminate chan struct{}
}

// NewOSThread creates a new thread.
//
// It is assumed that the OS thread is fixed by runtime.LockOSThread when NewOSThread is called.
func NewOSThread() *OSThread {
	return &OSThread{
		funcs:     make(chan func()),
		done:      make(chan struct{}),
		terminate: make(chan struct{}),
	}
}

// Loop starts the thread loop until Stop is called.
//
// Loop must be called on the thread.
func (t *OSThread) Loop() {
	for {
		select {
		case fn := <-t.funcs:
			func() {
				defer func() {
					t.done <- struct{}{}
				}()

				fn()
			}()
		case <-t.terminate:
			return
		}
	}
}

// Stop stops the thread loop.
func (t *OSThread) Stop() {
	t.terminate <- struct{}{}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call blocks if Loop is not called.
func (t *OSThread) Call(f func()) {
	t.funcs <- f
	<-t.done
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
func (t *NoopThread) Call(f func()) { f() }

// Stop does nothing
func (t *NoopThread) Stop() {}

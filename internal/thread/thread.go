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
	"runtime"
)

type Thread interface {
	Loop(ctx context.Context) error
	Call(f func())
	CallAsync(f func())

	private()
}

type queueItem struct {
	f    func()
	sync bool
}

// OSThread represents an OS thread.
type OSThread struct {
	funcs chan queueItem
	done  chan struct{}
}

// NewOSThread creates a new thread.
func NewOSThread() *OSThread {
	return &OSThread{
		funcs: make(chan queueItem),
		done:  make(chan struct{}),
	}
}

// Loop starts the thread loop until Stop is called on the current OS thread.
//
// Loop returns ctx's error if exists.
//
// Loop must be called on the OS thread.
func (t *OSThread) Loop(ctx context.Context) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case item := <-t.funcs:
			func() {
				if item.sync {
					defer func() {
						t.done <- struct{}{}
					}()
				}
				item.f()
			}()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Call calls f on the thread.
//
// Do not call Call from the same thread. Call would block forever.
//
// Call blocks if Loop is not called.
func (t *OSThread) Call(f func()) {
	t.funcs <- queueItem{f: f, sync: true}
	<-t.done
}

func (t *OSThread) private() {
}

// CallAsync tries to queue f.
// CallAsync returns immediately if f can be queued.
// CallAsync blocks if f cannot be queued.
//
// Do not call CallAsync from the same thread. CallAsync would block forever.
func (t *OSThread) CallAsync(f func()) {
	t.funcs <- queueItem{f: f, sync: false}
}

// NoopThread is used to disable threading.
type NoopThread struct{}

// NewNoopThread creates a new thread that does no threading.
func NewNoopThread() *NoopThread {
	return &NoopThread{}
}

// Loop does nothing.
func (t *NoopThread) Loop(ctx context.Context) error {
	return nil
}

// Call executes the func immediately.
func (t *NoopThread) Call(f func()) {
	f()
}

// CallAsync executes the func immediately.
func (t *NoopThread) CallAsync(f func()) {
	f()
}

func (t *NoopThread) private() {
}

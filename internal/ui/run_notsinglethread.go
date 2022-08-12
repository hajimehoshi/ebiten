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

//go:build !android && !ios && !js && !nintendosdk && !ebitenginesinglethread && !ebitensinglethread
// +build !android,!ios,!js,!nintendosdk,!ebitenginesinglethread,!ebitensinglethread

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func (u *userInterfaceImpl) Run(game Game) error {
	u.context = newContext(game)

	// Initialize the main thread first so the thread is available at u.run (#809).
	u.t = thread.NewOSThread()
	graphicscommand.SetRenderingThread(u.t)

	ch := make(chan error, 1)
	go func() {
		defer u.t.Stop()

		defer close(ch)

		var err error
		if u.t.Call(func() {
			err = u.init()
		}); err != nil {
			ch <- err
			return
		}

		if err := u.loop(); err != nil {
			ch <- err
			return
		}
	}()

	u.setRunning(true)
	u.t.Loop()
	u.setRunning(false)
	return <-ch
}

// runOnAnotherThreadFromMainThread is called from the main thread, and calls f on a new goroutine (thread).
// runOnAnotherThreadFromMainThread creates a new nested main thread and runs the run loop.
// u.t is updated to the new thread until runOnAnotherThreadFromMainThread is called.
//
// Inside f, another functions that must be called from the main thread can be called safely.
func (u *userInterfaceImpl) runOnAnotherThreadFromMainThread(f func()) {
	// As this function is called from the main thread, u.t should never be accessed and can be updated here.
	t := u.t
	defer func() {
		u.t = t
		graphicscommand.SetRenderingThread(t)
	}()

	u.t = thread.NewOSThread()
	graphicscommand.SetRenderingThread(u.t)

	go func() {
		defer u.t.Stop()
		f()
	}()
	u.t.Loop()
}

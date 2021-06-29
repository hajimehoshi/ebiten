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

//go:build !ebitensinglethread && (darwin || freebsd || linux || windows) && !android && !ios
// +build !ebitensinglethread
// +build darwin freebsd linux windows
// +build !android
// +build !ios

package glfw

import (
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func (u *UserInterface) Run(uicontext driver.UIContext) error {
	u.context = uicontext

	// Initialize the main thread first so the thread is available at u.run (#809).
	u.t = thread.NewOSThread()
	graphicscommand.SetMainThread(u.t)

	ch := make(chan error, 1)
	go func() {
		defer func() {
			_ = u.t.Call(func() error {
				return thread.BreakLoop
			})
		}()

		defer close(ch)

		if err := u.t.Call(func() error {
			return u.init()
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
func (u *UserInterface) runOnAnotherThreadFromMainThread(f func() error) error {
	// As this function is called from the main thread, u.t should never be accessed and can be updated here.
	t := u.t
	defer func() {
		u.t = t
		graphicscommand.SetMainThread(t)
	}()

	u.t = thread.NewOSThread()
	graphicscommand.SetMainThread(u.t)

	var err error
	go func() {
		defer func() {
			_ = u.t.Call(func() error {
				return thread.BreakLoop
			})
		}()
		err = f()
	}()
	u.t.Loop()
	return err
}

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
	u.t = thread.New()
	graphicscommand.SetMainThread(u.t)

	ch := make(chan error, 1)
	go func() {
		defer func() {
			_ = u.t.Call(func() error {
				return thread.BreakLoop
			})
		}()

		defer close(ch)

		_ = u.t.Call(func() error {
			if err := u.init(); err != nil {
				ch <- err
			}
			return nil
		})

		if err := u.loop(); err != nil {
			ch <- err
		}
	}()

	u.setRunning(true)
	u.t.Loop()
	u.setRunning(false)
	return <-ch
}

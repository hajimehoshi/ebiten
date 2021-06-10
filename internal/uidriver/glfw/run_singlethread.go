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

//go:build ebitensinglethread && (darwin || freebsd || linux || windows) && !android && !ios
// +build ebitensinglethread
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
	u.t = thread.NewNoopThread()
	graphicscommand.SetMainThread(u.t)

	u.setRunning(true)

	if err := u.init(); err != nil {
		return err
	}

	if err := u.loop(); err != nil {
		return err
	}

	u.setRunning(false)
	return nil
}

func (u *UserInterface) runOnAnotherThreadFromMainThread(f func() error) error {
	return f()
}

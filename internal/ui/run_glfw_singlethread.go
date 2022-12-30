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

//go:build !android && !ios && !js && !nintendosdk && (ebitenginesinglethread || ebitensinglethread)

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func (u *userInterfaceImpl) Run(game Game, options *RunOptions) error {
	u.context = newContext(game)

	// Initialize the main thread first so the thread is available at u.run (#809).
	u.mainThread = thread.NewNoopThread()
	u.renderThread = thread.NewNoopThread()
	graphicscommand.SetRenderThread(u.renderThread)

	u.setRunning(true)
	defer u.setRunning(false)

	if err := u.initOnMainThread(options); err != nil {
		return err
	}

	if err := u.loopGame(); err != nil {
		return err
	}

	return nil
}

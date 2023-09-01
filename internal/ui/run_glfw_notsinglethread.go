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

package ui

import (
	stdcontext "context"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func (u *userInterfaceImpl) Run(game Game, options *RunOptions) error {
	u.context = newContext(game)

	u.mainThread = thread.NewOSThread()
	u.renderThread = thread.NewOSThread()
	graphicscommand.SetRenderThread(u.renderThread)

	// Set the running state true after the main thread is set, and before initOnMainThread is called (#2742).
	// TODO: As the existance of the main thread is the same as the value of `running`, this is redundant.
	// Make `mainThread` atomic and remove `running` if possible.
	u.setRunning(true)
	defer u.setRunning(false)

	if err := u.initOnMainThread(options); err != nil {
		return err
	}

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the render thread.
	wg.Go(func() error {
		defer cancel()
		_ = u.renderThread.Loop(ctx)
		return nil
	})

	// Run the game thread.
	wg.Go(func() error {
		defer cancel()
		return u.loopGame()
	})

	// Run the main thread.
	_ = u.mainThread.Loop(ctx)
	return wg.Wait()
}

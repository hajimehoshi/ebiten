// Copyright 2023 The Ebitengine Authors
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

//go:build !android && !ios

package ui

import (
	stdcontext "context"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	if options.SingleThread || buildTagSingleThread || runtime.GOOS == "js" {
		return u.runSingleThread(game, options)
	}
	return u.runMultiThread(game, options)
}

func (u *UserInterface) runMultiThread(game Game, options *RunOptions) error {
	u.mainThread = thread.NewOSThread()
	graphicscommand.SetOSThreadAsRenderThread()

	u.context = newContext(game)

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the render thread.
	wg.Go(func() error {
		defer cancel()

		graphicscommand.LoopRenderThread(ctx)
		return nil
	})

	// Run the game thread.
	wg.Go(func() error {
		defer cancel()

		var err error
		u.mainThread.Call(func() {
			if err1 := u.initOnMainThread(options); err1 != nil {
				err = err1
			}
		})
		if err != nil {
			return err
		}

		// setRunning(true) should be called in initOnMainThread for each platform.
		defer u.setRunning(false)

		return u.loopGame()
	})

	// Run the main thread.
	_ = u.mainThread.Loop(ctx)
	return wg.Wait()
}

func (u *UserInterface) runSingleThread(game Game, options *RunOptions) error {
	// Initialize the main thread first so the thread is available at u.run (#809).
	u.mainThread = thread.NewNoopThread()

	u.setRunning(true)
	defer u.setRunning(false)

	u.context = newContext(game)

	if err := u.initOnMainThread(options); err != nil {
		return err
	}

	if err := u.loopGame(); err != nil {
		return err
	}

	return nil
}

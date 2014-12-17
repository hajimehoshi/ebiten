/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// A Game is the interface that represents a game.
type Game interface {
	Update() error
	Draw(gr GraphicsContext) error
}

var currentUI *ui

func init() {
	runtime.LockOSThread()
}

// Run runs the game.
// This function must be called from the main thread.
func Run(game Game, width, height, scale int, title string, fps int) error {
	ui, err := newUI(width, height, scale, title)
	if err != nil {
		return err
	}
	defer ui.terminate()

	currentUI = ui
	defer func() {
		currentUI = nil
	}()

	frameTime := time.Duration(int64(time.Second) / int64(fps))
	tick := time.Tick(frameTime)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	for {
		ui.doEvents()
		if ui.isClosed() {
			return nil
		}
		select {
		default:
			if err := ui.draw(game.Draw); err != nil {
				return err
			}
		case <-tick:
			if err := game.Update(); err != nil {
				return err
			}
		case <-sigterm:
			return nil
		}
	}
}

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
	"syscall"
	"time"
)

var currentUI *ui

// Run runs the game.
func Run(game Game, width, height, scale int, title string, fps int) error {
	ui := new(ui)

	currentUI = ui
	defer func() {
		currentUI = nil
	}()

	if err := ui.Start(game, width, height, scale, title); err != nil {
		return err
	}
	defer ui.Terminate()

	frameTime := time.Duration(int64(time.Second) / int64(fps))
	tick := time.Tick(frameTime)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	for {
		ui.DoEvents()
		if ui.IsClosed() {
			return nil
		}
		select {
		default:
			if err := ui.DrawGame(game); err != nil {
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

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

type Game interface {
	Update() error
	Draw(context GraphicsContext) error
}

// Run runs the game. Basically, this function executes ui.Start() at the start,
// calls ui.DoEvent(), game.Update() and game.Draw() at a regular interval, and finally
// calls ui.Terminate().
func Run(ui UI, game Game, width, height, scale int, title string, fps int) error {
	canvas, err := ui.Start(width, height, scale, title)
	if err != nil {
		return err
	}

	frameTime := time.Duration(int64(time.Second) / int64(fps))
	tick := time.Tick(frameTime)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	defer ui.Terminate()
	for {
		ui.DoEvents()
		select {
		default:
			if err := canvas.Draw(game); err != nil {
				return err
			}
		case <-tick:
			if err := game.Update(); err != nil {
				return err
			}
			if canvas.IsClosed() {
				return nil
			}
		case <-sigterm:
			return nil
		}
	}
}

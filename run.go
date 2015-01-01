// Copyright 2014 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"time"
)

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
//
// This function must be called from the main thread.
//
// The given function f is expected to be called 60 times a second,
// but this is not strictly guaranteed.
// If you need to care about time, you need to check current time every time f is called.
func Run(f func(*Image) error, width, height, scale int, title string) error {
	ui, err := ui.New(width, height, scale, title)
	if err != nil {
		return err
	}
	defer ui.Terminate()

	var graphicsContext *graphicsContext
	ui.Use(func(c *opengl.Context) {
		graphicsContext, err = newGraphicsContext(c, width, height, ui.ActualScale())
	})
	if err != nil {
		return err
	}

	for {
		// To avoid busy loop when the window is inactive, wait 1/120 [sec] at least.
		ch := time.After(1 * time.Second / 120)
		ui.DoEvents()
		if ui.IsClosed() {
			return nil
		}
		ui.Use(func(*opengl.Context) {
			err = graphicsContext.preUpdate()
		})
		if err != nil {
			return err
		}
		if err := f(&Image{ui: ui, inner: graphicsContext.screen}); err != nil {
			return err
		}
		ui.Use(func(*opengl.Context) {
			err = graphicsContext.postUpdate()
			if err != nil {
				return
			}
			ui.SwapBuffers()
		})
		if err != nil {
			return err
		}
		<-ch
	}
}

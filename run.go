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
	"github.com/hajimehoshi/ebiten/internal/audio"
	"github.com/hajimehoshi/ebiten/internal/graphics/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"time"
)

var fps = 0.0

// CurrentFPS returns the current number of frames per second.
func CurrentFPS() float64 {
	return fps
}

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
	actualScale, err := ui.Start(width, height, scale, title)
	if err != nil {
		return err
	}
	defer ui.Terminate()

	var graphicsContext *graphicsContext
	useGLContext(func(c *opengl.Context) {
		graphicsContext, err = newGraphicsContext(c, width, height, actualScale)
	})
	if err != nil {
		return err
	}

	frames := 0
	t := time.Now().UnixNano()
	for {
		if err := ui.DoEvents(); err != nil {
			return err
		}
		if ui.IsClosed() {
			return nil
		}
		if err := graphicsContext.preUpdate(); err != nil {
			return err
		}
		if err := f(graphicsContext.screen); err != nil {
			return err
		}
		if err := graphicsContext.postUpdate(); err != nil {
			return err
		}
		// TODO: I'm not sure this is 'Update'. Is 'Tick' better?
		audio.Update()
		ui.SwapBuffers()
		if err != nil {
			return err
		}

		// Calc the current FPS.
		now := time.Now().UnixNano()
		frames++
		if time.Second <= time.Duration(now-t) {
			fps = float64(frames) * float64(time.Second) / float64(now-t)
			t = now
			frames = 0
		}
	}
}

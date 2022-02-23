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
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type gameForUI struct {
	game      Game
	offscreen *Image
	screen    *Image
}

func newGameForUI(game Game) *gameForUI {
	return &gameForUI{
		game: game,
	}
}

func (c *gameForUI) Layout(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
	ow, oh := c.game.Layout(int(outsideWidth), int(outsideHeight))
	if ow <= 0 || oh <= 0 {
		panic("ebiten: Layout must return positive numbers")
	}

	sw, sh := int(outsideWidth*deviceScaleFactor), int(outsideHeight*deviceScaleFactor)
	if c.screen != nil {
		if w, h := c.screen.Size(); w != sw || h != sh {
			c.screen.Dispose()
			c.screen = nil
		}
	}
	if c.screen == nil {
		c.screen = newScreenFramebufferImage(sw, sh)
	}

	if c.offscreen != nil {
		if w, h := c.offscreen.Size(); w != ow || h != oh {
			c.offscreen.Dispose()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = NewImage(ow, oh)

		// Keep the offscreen an independent image from an atlas (#1938).
		// The shader program for the screen is special and doesn't work well with an image on an atlas.
		// An image on an atlas is surrounded by a transparent edge,
		// and the shader program unexpectedly picks the pixel on the edges.
		c.offscreen.mipmap.SetIndependent(true)
	}

	return ow, oh
}

func (c *gameForUI) Update() error {
	return c.game.Update()
}

func (c *gameForUI) Draw(screenScale float64, offsetX, offsetY float64, needsClearingScreen bool, framebufferYDirection graphicsdriver.YDirection, clearScreenEveryFrame, filterEnabled bool) error {
	c.offscreen.mipmap.SetVolatile(clearScreenEveryFrame)

	// Even though updateCount == 0, the offscreen is cleared and Draw is called.
	// Draw should not update the game state and then the screen should not be updated without Update, but
	// users might want to process something at Draw with the time intervals of FPS.
	if clearScreenEveryFrame {
		c.offscreen.Clear()
	}
	c.game.Draw(c.offscreen)

	if needsClearingScreen {
		// This clear is needed for fullscreen mode or some mobile platforms (#622).
		c.screen.Clear()
	}

	op := &DrawImageOptions{}

	s := screenScale
	switch framebufferYDirection {
	case graphicsdriver.Upward:
		op.GeoM.Scale(s, -s)
		_, h := c.offscreen.Size()
		op.GeoM.Translate(0, float64(h)*s)
	case graphicsdriver.Downward:
		op.GeoM.Scale(s, s)
	default:
		panic(fmt.Sprintf("ebiten: invalid v-direction: %d", framebufferYDirection))
	}

	op.GeoM.Translate(offsetX, offsetY)
	op.CompositeMode = CompositeModeCopy

	switch {
	case !filterEnabled:
		op.Filter = FilterNearest
	case math.Floor(s) == s:
		op.Filter = FilterNearest
	case s > 1:
		op.Filter = filterScreen
	default:
		// filterScreen works with >=1 scale, but does not well with <1 scale.
		// Use regular FilterLinear instead so far (#669).
		op.Filter = FilterLinear
	}
	c.screen.DrawImage(c.offscreen, op)
	return nil
}

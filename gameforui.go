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
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type gameForUI struct {
	game      Game
	offscreen *Image
}

func newGameForUI(game Game) *gameForUI {
	return &gameForUI{
		game: game,
	}
}

func (c *gameForUI) NewOffscreenImage(width, height int) *ui.Image {
	if c.offscreen != nil {
		c.offscreen.Dispose()
		c.offscreen = nil
	}

	// Keep the offscreen an unmanaged image that is always isolated from an atlas (#1938).
	// The shader program for the screen is special and doesn't work well with an image on an atlas.
	// An image on an atlas is surrounded by a transparent edge,
	// and the shader program unexpectedly picks the pixel on the edges.
	imageType := atlas.ImageTypeUnmanaged
	if ui.IsScreenClearedEveryFrame() {
		// A violatile image is also always isolated.
		imageType = atlas.ImageTypeVolatile
	}
	c.offscreen = newImage(image.Rect(0, 0, width, height), imageType)
	return c.offscreen.image
}

func (c *gameForUI) Layout(outsideWidth, outsideHeight int) (int, int) {
	return c.game.Layout(outsideWidth, outsideHeight)
}

func (c *gameForUI) Update() error {
	return c.game.Update()
}

func (c *gameForUI) Draw() {
	// TODO: This is a dirty hack to fix #2362. Move setVerticesCache to ui.Image if possible.
	c.offscreen.resolveSetVerticesCacheIfNeeded()
	c.game.Draw(c.offscreen)
}

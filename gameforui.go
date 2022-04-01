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
	c.offscreen = NewImage(width, height)
	return c.offscreen.image
}

func (c *gameForUI) Layout(outsideWidth, outsideHeight int) (int, int) {
	return c.game.Layout(outsideWidth, outsideHeight)
}

func (c *gameForUI) Update() error {
	return c.game.Update()
}

func (c *gameForUI) Draw() {
	c.game.Draw(c.offscreen)
}

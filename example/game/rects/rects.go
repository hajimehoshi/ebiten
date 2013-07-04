// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package rects

import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image/color"
	"math/rand"
	"time"
)

type Rects struct {
	rectsTexture graphics.Texture
}

func New() *Rects {
	return &Rects{}
}

func (game *Rects) ScreenWidth() int {
	return 256
}

func (game *Rects) ScreenHeight() int {
	return 240
}

func (game *Rects) Fps() int {
	return 60
}

func (game *Rects) Init(tf graphics.TextureFactory) {
	game.rectsTexture = tf.NewTexture(game.ScreenWidth(), game.ScreenHeight())
}

func (game *Rects) Update(inputState ebiten.InputState) {
}

func (game *Rects) Draw(g graphics.Context) {
	g.SetOffscreen(game.rectsTexture.ID)

	x := rand.Intn(game.ScreenWidth())
	y := rand.Intn(game.ScreenHeight())
	width := rand.Intn(game.ScreenWidth() - x)
	height := rand.Intn(game.ScreenHeight() - y)

	red := uint8(rand.Intn(256))
	green := uint8(rand.Intn(256))
	blue := uint8(rand.Intn(256))
	alpha := uint8(rand.Intn(256))

	g.DrawRect(
		graphics.Rect{x, y, width, height},
		&color.RGBA{red, green, blue, alpha},
	)

	g.SetOffscreen(g.Screen().ID)
	g.DrawTexture(game.rectsTexture.ID,
		matrix.IdentityGeometry(),
		matrix.IdentityColor())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

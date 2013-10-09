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

package rotating

import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
)

type Rotating struct {
	ebitenTexture  graphics.Texture
	x              int
	geometryMatrix matrix.Geometry
}

func New() *Rotating {
	return &Rotating{}
}

func (game *Rotating) Init(tf graphics.TextureFactory) {
	file, err := os.Open("images/ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.ebitenTexture, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Rotating) Update(context ebiten.GameContext) {
	game.x++

	game.geometryMatrix = matrix.IdentityGeometry()
	tx, ty := float64(game.ebitenTexture.Width()), float64(game.ebitenTexture.Height())
	game.geometryMatrix.Translate(-tx/2, -ty/2)
	game.geometryMatrix.Rotate(float64(game.x) * 2 * math.Pi / float64(ebiten.FPS*10))
	game.geometryMatrix.Translate(tx/2, ty/2)
	centerX := float64(context.ScreenWidth()) / 2
	centerY := float64(context.ScreenHeight()) / 2
	game.geometryMatrix.Translate(centerX-tx/2, centerY-ty/2)
}

func (game *Rotating) Draw(g graphics.Context) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})
	g.DrawTexture(game.ebitenTexture.ID(),
		game.geometryMatrix,
		matrix.IdentityColor())
}

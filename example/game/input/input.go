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

package input

import (
	"fmt"
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	"os"
)

type Input struct {
	textTexture graphics.Texture
	inputState  ebiten.InputState
}

func New() *Input {
	return &Input{}
}

func (game *Input) ScreenWidth() int {
	return 256
}

func (game *Input) ScreenHeight() int {
	return 240
}

func (game *Input) Fps() int {
	return 60
}

func (game *Input) Init(tf graphics.TextureFactory) {
	file, err := os.Open("images/text.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.textTexture, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Input) Update(context ebiten.GameContext) {
	game.inputState = context.InputState()
}

func (game *Input) Draw(g graphics.Context) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})
	str := fmt.Sprintf(`Input State:
  X: %d
  Y: %d`, game.inputState.X, game.inputState.Y)
	game.drawText(g, str, 5, 5)
}

func (game *Input) drawText(g graphics.Context, text string, x, y int) {
	const letterWidth = 6
	const letterHeight = 16

	parts := []graphics.TexturePart{}
	textX := 0
	textY := 0
	for _, c := range text {
		if c == '\n' {
			textX = 0
			textY += letterHeight
			continue
		}
		code := int(c)
		x := (code % 32) * letterWidth
		y := (code / 32) * letterHeight
		source := graphics.Rect{x, y, letterWidth, letterHeight}
		parts = append(parts, graphics.TexturePart{
			LocationX: textX,
			LocationY: textY,
			Source:    source,
		})
		textX += letterWidth
	}

	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Translate(float64(x), float64(y))
	colorMatrix := matrix.IdentityColor()
	g.DrawTextureParts(game.textTexture.ID(), parts,
		geometryMatrix, colorMatrix)
}

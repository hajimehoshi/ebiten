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

package monochrome

import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
	"os"
)

type Monochrome struct {
	ebitenTexture graphics.Texture
	ch            chan bool
	colorMatrix   matrix.Color
}

func New() *Monochrome {
	return &Monochrome{
		ch:          make(chan bool),
		colorMatrix: matrix.IdentityColor(),
	}
}

func (game *Monochrome) ScreenWidth() int {
	return 256
}

func (game *Monochrome) ScreenHeight() int {
	return 240
}

func (game *Monochrome) Fps() int {
	return 60
}

func (game *Monochrome) Init(tf graphics.TextureFactory) {
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

	go game.update()
}

func mean(a, b matrix.Color, k float64) matrix.Color {
	dim := a.Dim()
	result := matrix.Color{}
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			result.Elements[i][j] =
				a.Elements[i][j]*(1-k) +
					b.Elements[i][j]*k
		}
	}
	return result
}

func (game *Monochrome) update() {
	colorI := matrix.IdentityColor()
	colorMonochrome := matrix.Monochrome()
	for {
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			rate := float64(i) / float64(game.Fps())
			game.colorMatrix = mean(colorI, colorMonochrome, rate)
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			game.colorMatrix = colorMonochrome
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			rate := float64(i) / float64(game.Fps())
			game.colorMatrix = mean(colorMonochrome, colorI, rate)
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			game.colorMatrix = colorI
			game.ch <- true
		}
	}
}

func (game *Monochrome) Update(inputState ebiten.InputState) {
	game.ch <- true
	<-game.ch
}

func (game *Monochrome) Draw(g graphics.Context) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx := game.ScreenWidth()/2 - game.ebitenTexture.Width/2
	ty := game.ScreenHeight()/2 - game.ebitenTexture.Height/2
	geometryMatrix.Translate(float64(tx), float64(ty))
	g.DrawTexture(game.ebitenTexture.ID,
		geometryMatrix, game.colorMatrix)
}

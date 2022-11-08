// Copyright 2022 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

var (
	whiteImage        = ebiten.NewImage(3, 3)
	debugCircleImage  *ebiten.Image
	whiteTextureImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	face              font.Face
)

func init() {
	whiteImage.Fill(color.White)

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	debugCircleImage = ebiten.NewImageFromImage(img)

	whiteImage.Fill(color.Black)

	f, _ := opentype.Parse(goregular.TTF)
	face, _ = opentype.NewFace(f, &opentype.FaceOptions{
		Size: 12,
		DPI:  72,
	})
}

type Game struct {
	counter int
}

func (g *Game) Update() error {
	g.counter++
	if g.counter > 16 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Before the fix, some complex renderings with EvenOdd might cause a DirectX error like this (#2138):
	//     panic: directx: IDXGISwapChain4::Present failed: HRESULT(2289696773)

	screen.DrawImage(debugCircleImage, nil)
	text.Draw(screen, "014678.,", face, 100, 100, color.White)

	p := vector.Path{}
	p.Arc(100, 100, 6, 0, 2*math.Pi, vector.Clockwise)
	filling, indicies := p.AppendVerticesAndIndicesForFilling(nil, nil)
	screen.DrawTriangles(filling, indicies, whiteTextureImage, &ebiten.DrawTrianglesOptions{
		FillRule: ebiten.EvenOdd,
	})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}

}

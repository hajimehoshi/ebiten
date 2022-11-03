// Copyright 2018 The Ebiten Authors
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

package main

import (
	"bytes"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var ebitenImage *ebiten.Image

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)
}

func drawRect(screen *ebiten.Image, img *ebiten.Image, x, y, width, height float32, address ebiten.Address, msg string) {
	sx, sy := -width/2, -height/2
	vs := []ebiten.Vertex{
		{
			DstX:   x,
			DstY:   y,
			SrcX:   sx,
			SrcY:   sy,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x + width,
			DstY:   y,
			SrcX:   sx + width,
			SrcY:   sy,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x,
			DstY:   y + height,
			SrcX:   sx,
			SrcY:   sy + height,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x + width,
			DstY:   y + height,
			SrcX:   sx + width,
			SrcY:   sy + height,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = address
	screen.DrawTriangles(vs, []uint16{0, 1, 2, 1, 2, 3}, img, op)

	ebitenutil.DebugPrintAt(screen, msg, int(x), int(y)-16)
}

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	const ox, oy = 40, 60
	drawRect(screen, ebitenImage, ox, oy, 200, 100, ebiten.AddressClampToZero, "Regular")
	drawRect(screen, ebitenImage, 220+ox, oy, 200, 100, ebiten.AddressRepeat, "Regular, Repeat")

	subImage := ebitenImage.SubImage(image.Rect(10, 5, 20, 30)).(*ebiten.Image)
	drawRect(screen, subImage, ox, 200+oy, 200, 100, ebiten.AddressClampToZero, "Subimage")
	drawRect(screen, subImage, 220+ox, 200+oy, 200, 100, ebiten.AddressRepeat, "Subimage, Repeat")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Sampler Address (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

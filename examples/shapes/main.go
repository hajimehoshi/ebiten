// Copyright 2017 The Ebiten Authors
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

// +build example jsgo

package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	emptyImage, _ = ebiten.NewImage(16, 16, ebiten.FilterDefault)
)

func init() {
	emptyImage.Fill(color.White)
}

var count = 0

func line(x0, y0, x1, y1 float32, clr color.RGBA) ([]ebiten.Vertex, []uint16) {
	const width = 0.5

	theta := math.Atan2(float64(x1-x0), float64(y1-y0))
	dx := float32(math.Cos(theta) - math.Sin(theta))
	dy := float32(math.Sin(theta) + math.Cos(theta))

	r := float32(clr.R) / 0xff
	g := float32(clr.G) / 0xff
	b := float32(clr.B) / 0xff
	a := float32(clr.A) / 0xff

	return []ebiten.Vertex{
		{
			DstX:   x0 - width*dx,
			DstY:   y0 - width*dy,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x0 + width*dx,
			DstY:   y0 + width*dy,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 - width*dx,
			DstY:   y1 - width*dy,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 + width*dx,
			DstY:   y1 + width*dy,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	}, []uint16{0, 1, 2, 1, 2, 3}
}

func update(screen *ebiten.Image) error {
	count++
	count %= 240

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	cf := float64(count)
	v, i := line(100, 100, 300, 100, color.RGBA{0xff, 0xff, 0xff, 0xff})
	screen.DrawTriangles(v, i, emptyImage, nil)
	v, i = line(50, 150, 50, 350, color.RGBA{0xff, 0xff, 0xff, 0xff})
	screen.DrawTriangles(v, i, emptyImage, nil)
	v, i = line(50, 100+float32(cf), 200+float32(cf), 250, color.RGBA{0x00, 0xff, 0xff, 0xff})
	screen.DrawTriangles(v, i, emptyImage, nil)
	ebitenutil.DrawRect(screen, 50+cf, 50+cf, 100+cf, 100+cf, color.RGBA{0x80, 0x80, 0x80, 0x80})
	ebitenutil.DrawRect(screen, 300-cf, 50, 120, 120, color.RGBA{0x00, 0x80, 0x00, 0x80})

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Shapes (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

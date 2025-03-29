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
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	whiteImage = ebiten.NewImage(3, 3)
)

func init() {
	whiteImage.Fill(color.White)
}

func genVertices(num int) []ebiten.Vertex {
	const (
		centerX = screenWidth / 2
		centerY = screenHeight / 2
		r       = 160
	)

	vs := []ebiten.Vertex{}
	for i := 0; i < num; i++ {
		rate := float64(i) / float64(num)
		cr := 0.0
		cg := 0.0
		cb := 0.0
		if rate < 1.0/3.0 {
			cb = 2 - 2*(rate*3)
			cr = 2 * (rate * 3)
		}
		if 1.0/3.0 <= rate && rate < 2.0/3.0 {
			cr = 2 - 2*(rate-1.0/3.0)*3
			cg = 2 * (rate - 1.0/3.0) * 3
		}
		if 2.0/3.0 <= rate {
			cg = 2 - 2*(rate-2.0/3.0)*3
			cb = 2 * (rate - 2.0/3.0) * 3
		}
		vs = append(vs, ebiten.Vertex{
			DstX:   float32(r*math.Cos(2*math.Pi*rate)) + centerX,
			DstY:   float32(r*math.Sin(2*math.Pi*rate)) + centerY,
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(cr),
			ColorG: float32(cg),
			ColorB: float32(cb),
			ColorA: 1,
		})
	}

	vs = append(vs, ebiten.Vertex{
		DstX:   centerX,
		DstY:   centerY,
		SrcX:   0,
		SrcY:   0,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	})

	return vs
}

type Game struct {
	debugui debugui.DebugUI

	vertices []ebiten.Vertex

	ngon     int
	prevNgon int
}

func (g *Game) Update() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Polygons", image.Rect(10, 10, 210, 110), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
			ctx.Slider(&g.ngon, 1, 40, 1)
		})
		return nil
	}); err != nil {
		return err
	}

	if g.prevNgon != g.ngon || len(g.vertices) == 0 {
		g.vertices = genVertices(g.ngon)
		g.prevNgon = g.ngon
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = ebiten.AddressUnsafe
	indices := []uint16{}
	for i := 0; i < g.ngon; i++ {
		indices = append(indices, uint16(i), uint16(i+1)%uint16(g.ngon), uint16(g.ngon))
	}
	screen.DrawTriangles(g.vertices, indices, whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image), op)

	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Polygons (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{ngon: 10}); err != nil {
		log.Fatal(err)
	}
}

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

// +build example jsgo

package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
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

var (
	vertices []ebiten.Vertex

	ngon     = 10
	prevNgon = 0
)

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

func update(screen *ebiten.Image) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		ngon--
		if ngon < 1 {
			ngon = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		ngon++
		if ngon > 120 {
			ngon = 120
		}
	}

	if prevNgon != ngon || len(vertices) == 0 {
		vertices = genVertices(ngon)
		prevNgon = ngon
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	op := &ebiten.DrawTrianglesOptions{}
	indices := []uint16{}
	for i := 0; i < ngon; i++ {
		indices = append(indices, uint16(i), uint16(i+1)%uint16(ngon), uint16(ngon))
	}
	screen.DrawTriangles(vertices, indices, emptyImage, op)

	msg := fmt.Sprintf("TPS: %0.2f\n%d-gon\nPress <- or -> to change the number of the vertices", ebiten.CurrentTPS(), ngon)
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Polygons (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

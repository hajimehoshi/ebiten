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

	numVerticesToDraw     = 10
	prevNumVerticesToDraw = 0
)

func genVertices(num int) []ebiten.Vertex {
	const (
		centerX = screenWidth / 2
		centerY = screenHeight / 2
		r       = 160
	)

	vs := []ebiten.Vertex{}
	for i := 0; i < num; i++ {
		theta := float64(i) / float64(num) * 2 * math.Pi
		cr := float32(0)
		cg := float32(0)
		cb := float32(0)
		if 0 <= i && i < 2*num/3 {
			cr = 2 * float32(i) / float32(num/3)
		}
		if num/3 <= i && i < 2*num/3 {
			cr = 2 - 2*float32(i-num/3)/float32(num/3)
		}
		if num/3 <= i && i < 2*num/3 {
			cg = 2 * float32(i-num/3) / float32(num/3)
		}
		if 2*num/3 <= i && i < num {
			cg = 2 - 2*float32(i-2*num/3)/float32(num/3)
		}
		if 2*num/3 <= i && i < num {
			cb = 2 * float32(i-2*num/3) / float32(num/3)
		}
		if 0 <= i && i < num/3 {
			cb = 2 - 2*float32(i)/float32(num/3)
		}
		vs = append(vs, ebiten.Vertex{
			DstX:   float32(r*math.Cos(theta)) + centerX,
			DstY:   float32(r*math.Sin(theta)) + centerY,
			SrcX:   0,
			SrcY:   0,
			ColorR: cr,
			ColorG: cg,
			ColorB: cb,
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
		numVerticesToDraw--
		if numVerticesToDraw < 1 {
			numVerticesToDraw = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		numVerticesToDraw++
		if numVerticesToDraw > 120 {
			numVerticesToDraw = 120
		}
	}

	if prevNumVerticesToDraw != numVerticesToDraw || len(vertices) == 0 {
		vertices = genVertices(numVerticesToDraw)
		prevNumVerticesToDraw = numVerticesToDraw
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	op := &ebiten.DrawTrianglesOptions{}
	indices := []uint16{}
	for i := 0; i < numVerticesToDraw; i++ {
		indices = append(indices, uint16(i), uint16(i+1)%uint16(numVerticesToDraw), uint16(numVerticesToDraw))
	}
	screen.DrawTriangles(vertices, indices, emptyImage, op)

	msg := fmt.Sprintf("TPS: %0.2f\nVertices: %d\nPress <- or -> to change the number of the vertices", ebiten.CurrentTPS(), numVerticesToDraw)
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Triangles (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

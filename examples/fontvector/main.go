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

package main

import (
	"bytes"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	whiteImage = ebiten.NewImage(3, 3)

	// whiteSubImage is an internal sub image of whiteImage.
	// Use whiteSubImage at DrawTriangles instead of whiteImage in order to avoid bleeding edges.
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	whiteImage.Fill(color.White)
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	path     vector.Path
	vertices []ebiten.Vertex
	indices  []uint16

	tick int
}

func (g *Game) Update() error {
	if g.tick == 0 {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
		if err != nil {
			return err
		}
		op := &text.LayoutOptions{}
		op.LineSpacingInPixels = 110
		text.AppendVectorPath(&g.path, "ABCEDFG\nabcdefg\nあいうえお\nかきくけこ", &text.GoTextFace{
			Source: s,
			Size:   90,
		}, op)
	}

	g.tick++

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	op := &vector.StrokeOptions{}
	op.Width = 2*(float32(math.Sin(float64(g.tick)*2*math.Pi/180))+1) + 1
	op.LineJoin = vector.LineJoinRound
	op.LineCap = vector.LineCapRound
	g.vertices, g.indices = g.path.AppendVerticesAndIndicesForStroke(g.vertices, g.indices, op)

	for i := range g.vertices {
		g.vertices[i].DstX += 50
		g.vertices[i].DstY += 0
		g.vertices[i].SrcX = 1
		g.vertices[i].SrcY = 1
	}

	screen.DrawTriangles(g.vertices, g.indices, whiteSubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

func (*Game) Layout(width, height int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowTitle("Font Vector (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

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
	"image"
	"image/color"
	"log"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
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

func fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return float32(x>>6) + float32(x&((1<<6)-1))/float32(1<<6)
}

type Game struct {
	segments sfnt.Segments
	bounds   fixed.Rectangle26_6
	vertices []ebiten.Vertex
	indices  []uint16

	tick int
}

func (g *Game) Update() error {
	g.tick++

	if g.segments == nil {
		ppem := fixed.I(300)

		f, err := sfnt.Parse(fonts.MPlus1pRegular_ttf)
		if err != nil {
			return err
		}

		var b sfnt.Buffer
		idx, err := f.GlyphIndex(&b, 'あ')
		if err != nil {
			return err
		}

		segments, err := f.LoadGlyph(&b, idx, ppem, nil)
		if err != nil {
			return err
		}
		g.segments = segments

		bounds, err := f.Bounds(&b, ppem, font.HintingNone)
		if err != nil {
			return err
		}
		g.bounds = bounds
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	var path vector.Path
	for _, seg := range g.segments {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			path.MoveTo(
				fixed26_6ToFloat32(seg.Args[0].X),
				fixed26_6ToFloat32(seg.Args[0].Y),
			)
		case sfnt.SegmentOpLineTo:
			path.LineTo(
				fixed26_6ToFloat32(seg.Args[0].X),
				fixed26_6ToFloat32(seg.Args[0].Y),
			)
		case sfnt.SegmentOpQuadTo:
			path.QuadTo(
				fixed26_6ToFloat32(seg.Args[0].X),
				fixed26_6ToFloat32(seg.Args[0].Y),
				fixed26_6ToFloat32(seg.Args[1].X),
				fixed26_6ToFloat32(seg.Args[1].Y),
			)
		case sfnt.SegmentOpCubeTo:
			path.CubicTo(
				fixed26_6ToFloat32(seg.Args[0].X),
				fixed26_6ToFloat32(seg.Args[0].Y),
				fixed26_6ToFloat32(seg.Args[1].X),
				fixed26_6ToFloat32(seg.Args[1].Y),
				fixed26_6ToFloat32(seg.Args[2].X),
				fixed26_6ToFloat32(seg.Args[2].Y),
			)
		}
	}
	path.Close()

	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	op := &vector.StrokeOptions{}
	op.Width = 7*(float32(math.Sin(float64(g.tick)*2*math.Pi/180))+1) + 1
	op.LineJoin = vector.LineJoinRound
	op.LineCap = vector.LineCapRound
	g.vertices, g.indices = path.AppendVerticesAndIndicesForStroke(g.vertices, g.indices, op)

	for i := range g.vertices {
		g.vertices[i].DstX += screenWidth/2 - fixed26_6ToFloat32(g.bounds.Max.X+g.bounds.Min.X)/2
		g.vertices[i].DstY += screenHeight/2 - fixed26_6ToFloat32(g.bounds.Max.Y+g.bounds.Min.Y)/2
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

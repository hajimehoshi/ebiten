// Copyright 2019 The Ebiten Authors
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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

func drawEbitenText(screen *ebiten.Image, x, y int, aa bool, line bool) {
	var path vector.Path

	// E
	path.MoveTo(20, 20)
	path.LineTo(20, 70)
	path.LineTo(70, 70)
	path.LineTo(70, 60)
	path.LineTo(30, 60)
	path.LineTo(30, 50)
	path.LineTo(70, 50)
	path.LineTo(70, 40)
	path.LineTo(30, 40)
	path.LineTo(30, 30)
	path.LineTo(70, 30)
	path.LineTo(70, 20)
	path.Close()

	// B
	path.MoveTo(80, 20)
	path.LineTo(80, 70)
	path.LineTo(100, 70)
	path.QuadTo(150, 57.5, 100, 45)
	path.QuadTo(150, 32.5, 100, 20)
	path.Close()

	// I
	path.MoveTo(140, 20)
	path.LineTo(140, 70)
	path.LineTo(150, 70)
	path.LineTo(150, 20)
	path.Close()

	// T
	path.MoveTo(160, 20)
	path.LineTo(160, 30)
	path.LineTo(180, 30)
	path.LineTo(180, 70)
	path.LineTo(190, 70)
	path.LineTo(190, 30)
	path.LineTo(210, 30)
	path.LineTo(210, 20)
	path.Close()

	// E
	path.MoveTo(220, 20)
	path.LineTo(220, 70)
	path.LineTo(270, 70)
	path.LineTo(270, 60)
	path.LineTo(230, 60)
	path.LineTo(230, 50)
	path.LineTo(270, 50)
	path.LineTo(270, 40)
	path.LineTo(230, 40)
	path.LineTo(230, 30)
	path.LineTo(270, 30)
	path.LineTo(270, 20)
	path.Close()

	// N
	path.MoveTo(280, 20)
	path.LineTo(280, 70)
	path.LineTo(290, 70)
	path.LineTo(290, 35)
	path.LineTo(320, 70)
	path.LineTo(330, 70)
	path.LineTo(330, 20)
	path.LineTo(320, 20)
	path.LineTo(320, 55)
	path.LineTo(290, 20)
	path.Close()

	var vs []ebiten.Vertex
	var is []uint16
	if line {
		op := &vector.StrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		vs, is = path.AppendVerticesAndIndicesForStroke(nil, nil, op)
	} else {
		vs, is = path.AppendVerticesAndIndicesForFilling(nil, nil)
	}

	for i := range vs {
		vs[i].DstX = (vs[i].DstX + float32(x))
		vs[i].DstY = (vs[i].DstY + float32(y))
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 0xdb / float32(0xff)
		vs[i].ColorG = 0x56 / float32(0xff)
		vs[i].ColorB = 0x20 / float32(0xff)
		vs[i].ColorA = 1
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = aa
	if !line {
		// ebiten.EvenOdd is also fine here.
		// NonZero and EvenOdd differ when rendering a complex polygons with self-intersections and/or holes.
		// See https://en.wikipedia.org/wiki/Nonzero-rule and https://en.wikipedia.org/wiki/Even%E2%80%93odd_rule .
		op.FillRule = ebiten.NonZero
	}
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func drawEbitenLogo(screen *ebiten.Image, x, y int, aa bool, line bool) {
	const unit = 16

	var path vector.Path

	// TODO: Add curves
	path.MoveTo(0, 4*unit)
	path.LineTo(0, 6*unit)
	path.LineTo(2*unit, 6*unit)
	path.LineTo(2*unit, 5*unit)
	path.LineTo(3*unit, 5*unit)
	path.LineTo(3*unit, 4*unit)
	path.LineTo(4*unit, 4*unit)
	path.LineTo(4*unit, 2*unit)
	path.LineTo(6*unit, 2*unit)
	path.LineTo(6*unit, 1*unit)
	path.LineTo(5*unit, 1*unit)
	path.LineTo(5*unit, 0)
	path.LineTo(4*unit, 0)
	path.LineTo(4*unit, 2*unit)
	path.LineTo(2*unit, 2*unit)
	path.LineTo(2*unit, 3*unit)
	path.LineTo(unit, 3*unit)
	path.LineTo(unit, 4*unit)
	path.Close()

	var vs []ebiten.Vertex
	var is []uint16
	if line {
		op := &vector.StrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		vs, is = path.AppendVerticesAndIndicesForStroke(nil, nil, op)
	} else {
		vs, is = path.AppendVerticesAndIndicesForFilling(nil, nil)
	}

	for i := range vs {
		vs[i].DstX = (vs[i].DstX + float32(x))
		vs[i].DstY = (vs[i].DstY + float32(y))
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 0xdb / float32(0xff)
		vs[i].ColorG = 0x56 / float32(0xff)
		vs[i].ColorB = 0x20 / float32(0xff)
		vs[i].ColorA = 1
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = aa
	if !line {
		op.FillRule = ebiten.NonZero
	}
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func drawArc(screen *ebiten.Image, count int, aa bool, line bool) {
	var path vector.Path

	path.MoveTo(350, 100)
	const cx, cy, r = 450, 100, 70
	theta1 := math.Pi * float64(count) / 180
	x := cx + r*math.Cos(theta1)
	y := cy + r*math.Sin(theta1)
	path.ArcTo(450, 100, float32(x), float32(y), 30)
	path.LineTo(float32(x), float32(y))

	theta2 := math.Pi * float64(count) / 180 / 3
	path.MoveTo(550, 100)
	path.Arc(550, 100, 50, float32(theta1), float32(theta2), vector.Clockwise)
	path.Close()

	var vs []ebiten.Vertex
	var is []uint16
	if line {
		op := &vector.StrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		vs, is = path.AppendVerticesAndIndicesForStroke(nil, nil, op)
	} else {
		vs, is = path.AppendVerticesAndIndicesForFilling(nil, nil)
	}

	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 0x33 / float32(0xff)
		vs[i].ColorG = 0xcc / float32(0xff)
		vs[i].ColorB = 0x66 / float32(0xff)
		vs[i].ColorA = 1
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = aa
	if !line {
		op.FillRule = ebiten.NonZero
	}
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func maxCounter(index int) int {
	return 128 + (17*index+32)%64
}

func drawWave(screen *ebiten.Image, counter int, aa bool, line bool) {
	var path vector.Path

	const npoints = 8
	indexToPoint := func(i int, counter int) (float32, float32) {
		x, y := float32(i*screenWidth/(npoints-1)), float32(screenHeight/2)
		y += float32(30 * math.Sin(float64(counter)*2*math.Pi/float64(maxCounter(i))))
		return x, y
	}

	for i := 0; i <= npoints; i++ {
		if i == 0 {
			path.MoveTo(indexToPoint(i, counter))
			continue
		}
		cpx0, cpy0 := indexToPoint(i-1, counter)
		x, y := indexToPoint(i, counter)
		cpx1, cpy1 := x, y
		cpx0 += 30
		cpx1 -= 30
		path.CubicTo(cpx0, cpy0, cpx1, cpy1, x, y)
	}
	path.LineTo(screenWidth, screenHeight)
	path.LineTo(0, screenHeight)

	var vs []ebiten.Vertex
	var is []uint16
	if line {
		op := &vector.StrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		vs, is = path.AppendVerticesAndIndicesForStroke(nil, nil, op)
	} else {
		vs, is = path.AppendVerticesAndIndicesForFilling(nil, nil)
	}

	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 0x33 / float32(0xff)
		vs[i].ColorG = 0x66 / float32(0xff)
		vs[i].ColorB = 0xff / float32(0xff)
		vs[i].ColorA = 1
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = aa
	if !line {
		op.FillRule = ebiten.NonZero
	}
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

type Game struct {
	counter int

	aa   bool
	line bool
}

func (g *Game) Update() error {
	g.counter++

	// Switch anti-alias.
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.aa = !g.aa
	}

	// Switch lines.
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.line = !g.line
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	dst := screen

	dst.Fill(color.RGBA{0xe0, 0xe0, 0xe0, 0xff})
	drawEbitenText(dst, 0, 50, g.aa, g.line)
	drawEbitenLogo(dst, 20, 150, g.aa, g.line)
	drawArc(dst, g.counter, g.aa, g.line)
	drawWave(dst, g.counter, g.aa, g.line)

	msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS())
	msg += "\nPress A to switch anti-alias."
	msg += "\nPress L to switch the fill mode and the line mode."
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{counter: 0}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Vector (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

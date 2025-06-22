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

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	debugui debugui.DebugUI

	counter int

	aa   bool
	line bool

	paths      []*vector.Path
	pathColors []color.Color
}

func (g *Game) appendPathsForEbitenText(paths []*vector.Path, pathColors []color.Color, x, y int, line bool) ([]*vector.Path, []color.Color) {
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

	var newPath *vector.Path
	if line {
		op := &vector.AddPathStrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		newPath = &vector.Path{}
		newPath.AddPathStroke(&path, op)
	} else {
		newPath = &path
	}
	paths = append(paths, newPath)
	pathColors = append(pathColors, color.RGBA{0xdb, 0x56, 0x20, 0xff})
	return paths, pathColors
}

func (g *Game) appendPathsForEbitenLogo(paths []*vector.Path, pathColors []color.Color, x, y int, line bool) ([]*vector.Path, []color.Color) {
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

	var geoM ebiten.GeoM
	geoM.Translate(float64(x), float64(y))

	var newPath vector.Path
	if line {
		op := &vector.AddPathStrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		op.GeoM = geoM
		newPath.AddPathStroke(&path, op)
	} else {
		op := &vector.AddPathOptions{}
		op.GeoM = geoM
		newPath.AddPath(&path, op)
	}
	paths = append(paths, &newPath)
	pathColors = append(pathColors, color.RGBA{0xdb, 0x56, 0x20, 0xff})
	return paths, pathColors
}

func (g *Game) appendPathsForArc(paths []*vector.Path, pathColors []color.Color, count int, line bool) ([]*vector.Path, []color.Color) {
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

	var newPath *vector.Path
	if line {
		op := &vector.AddPathStrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		newPath = &vector.Path{}
		newPath.AddPathStroke(&path, op)
	} else {
		newPath = &path
	}
	paths = append(paths, newPath)
	pathColors = append(pathColors, color.RGBA{0x33, 0xcc, 0x66, 0xff})
	return paths, pathColors
}

func maxCounter(index int) int {
	return 128 + (17*index+32)%64
}

func (g *Game) appendPathsForWave(paths []*vector.Path, pathColors []color.Color, counter int, line bool) ([]*vector.Path, []color.Color) {
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

	var newPath *vector.Path
	if line {
		op := &vector.AddPathStrokeOptions{}
		op.Width = 5
		op.LineJoin = vector.LineJoinRound
		newPath = &vector.Path{}
		newPath.AddPathStroke(&path, op)
	} else {
		newPath = &path
	}
	paths = append(paths, newPath)
	pathColors = append(pathColors, color.RGBA{0x33, 0x66, 0xff, 0xff})
	return paths, pathColors
}

func (g *Game) Update() error {
	g.counter++

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Vector", image.Rect(10, screenHeight-160, 210, screenHeight-10), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
			ctx.Text(fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
			ctx.Checkbox(&g.aa, "Anti-alias")
			ctx.Checkbox(&g.line, "Line")
		})
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	dst := screen

	dst.Fill(color.RGBA{0xe0, 0xe0, 0xe0, 0xff})

	g.paths = g.paths[:0]
	g.pathColors = g.pathColors[:0]
	g.paths, g.pathColors = g.appendPathsForEbitenText(g.paths, g.pathColors, 0, 50, g.line)
	g.paths, g.pathColors = g.appendPathsForEbitenLogo(g.paths, g.pathColors, 20, 150, g.line)
	g.paths, g.pathColors = g.appendPathsForArc(g.paths, g.pathColors, g.counter, g.line)
	g.paths, g.pathColors = g.appendPathsForWave(g.paths, g.pathColors, g.counter, g.line)

	vector.FillPaths(screen, g.paths, g.pathColors, g.aa, vector.FillRuleNonZero)

	g.debugui.Draw(screen)
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

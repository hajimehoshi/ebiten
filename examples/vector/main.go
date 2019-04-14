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

// +build example jsgo

package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

func drawEbitenText(screen *ebiten.Image) {
	var path vector.Path

	// E
	path.MoveTo(60, 20)
	path.LineTo(20, 20)
	path.LineTo(20, 60)
	path.LineTo(60, 60)
	path.MoveTo(20, 40)
	path.LineTo(60, 40)

	// B
	path.MoveTo(110, 20)
	path.LineTo(80, 20)
	path.LineTo(80, 60)
	path.LineTo(110, 60)
	path.LineTo(120, 50)
	path.LineTo(110, 40)
	path.LineTo(120, 30)
	path.LineTo(110, 20)
	path.LineTo(80, 20)
	path.MoveTo(80, 40)
	path.LineTo(110, 40)

	// I
	path.MoveTo(140, 20)
	path.LineTo(140, 60)

	// T
	path.MoveTo(160, 20)
	path.LineTo(200, 20)
	path.MoveTo(180, 20)
	path.LineTo(180, 60)

	// E
	path.MoveTo(260, 20)
	path.LineTo(220, 20)
	path.LineTo(220, 60)
	path.LineTo(260, 60)
	path.MoveTo(220, 40)
	path.LineTo(260, 40)

	// N
	path.MoveTo(280, 60)
	path.LineTo(280, 20)
	path.LineTo(320, 60)
	path.LineTo(320, 20)

	op := &vector.DrawPathOptions{}
	op.LineWidth = 8
	op.StrokeColor = color.RGBA{0xdb, 0x56, 0x20, 0xff}
	path.Draw(screen, op)
}

type roundingPoint struct {
	cx     float32
	cy     float32
	r      float32
	degree int
}

func (r *roundingPoint) Update() {
	r.degree++
	r.degree %= 360
}

func (r *roundingPoint) Position() (float32, float32) {
	s, c := math.Sincos(float64(r.degree) / 360 * 2 * math.Pi)
	return r.cx + r.r*float32(c), r.cy + r.r*float32(s)
}

func drawLinesByRoundingPoints(screen *ebiten.Image, points []*roundingPoint) {
	if len(points) == 0 {
		return
	}

	var path vector.Path

	path.MoveTo(points[0].Position())
	for i := 1; i < len(points); i++ {
		path.LineTo(points[i].Position())
	}

	op := &vector.DrawPathOptions{}
	op.LineWidth = 4
	op.StrokeColor = color.White
	path.Draw(screen, op)
}

var points = []*roundingPoint{
	{
		cx:     100,
		cy:     120,
		r:      10,
		degree: 0,
	},
	{
		cx:     120,
		cy:     120,
		r:      10,
		degree: 90,
	},
	{
		cx:     100,
		cy:     140,
		r:      10,
		degree: 180,
	},
	{
		cx:     120,
		cy:     140,
		r:      10,
		degree: 270,
	},
}

func update(screen *ebiten.Image) error {
	for _, p := range points {
		p.Update()
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	drawEbitenText(screen)
	drawLinesByRoundingPoints(screen, points)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Vector (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

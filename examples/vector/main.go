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
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

func drawEbitenText(screen *ebiten.Image) {
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

	// B
	path.MoveTo(80, 20)
	path.LineTo(80, 70)
	path.LineTo(100, 70)
	path.QuadTo(150, 57.5, 100, 45)
	path.QuadTo(150, 32.5, 100, 20)

	// I
	path.MoveTo(140, 20)
	path.LineTo(140, 70)
	path.LineTo(150, 70)
	path.LineTo(150, 20)

	// T
	path.MoveTo(160, 20)
	path.LineTo(160, 30)
	path.LineTo(180, 30)
	path.LineTo(180, 70)
	path.LineTo(190, 70)
	path.LineTo(190, 30)
	path.LineTo(210, 30)
	path.LineTo(210, 20)

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

	op := &vector.FillOptions{
		Color: color.RGBA{0xdb, 0x56, 0x20, 0xff},
	}
	path.Fill(screen, op)
}

func drawEbitenLogo(screen *ebiten.Image, x, y int) {
	const unit = 16

	var path vector.Path
	xf, yf := float32(x), float32(y)

	// TODO: Add curves
	path.MoveTo(xf, yf+4*unit)
	path.LineTo(xf, yf+6*unit)
	path.LineTo(xf+2*unit, yf+6*unit)
	path.LineTo(xf+2*unit, yf+5*unit)
	path.LineTo(xf+3*unit, yf+5*unit)
	path.LineTo(xf+3*unit, yf+4*unit)
	path.LineTo(xf+4*unit, yf+4*unit)
	path.LineTo(xf+4*unit, yf+2*unit)
	path.LineTo(xf+6*unit, yf+2*unit)
	path.LineTo(xf+6*unit, yf+1*unit)
	path.LineTo(xf+5*unit, yf+1*unit)
	path.LineTo(xf+5*unit, yf)
	path.LineTo(xf+4*unit, yf)
	path.LineTo(xf+4*unit, yf+2*unit)
	path.LineTo(xf+2*unit, yf+2*unit)
	path.LineTo(xf+2*unit, yf+3*unit)
	path.LineTo(xf+unit, yf+3*unit)
	path.LineTo(xf+unit, yf+4*unit)

	op := &vector.FillOptions{
		Color: color.RGBA{0xdb, 0x56, 0x20, 0xff},
	}
	path.Fill(screen, op)
}

func maxCounter(index int) int {
	return 128 + (17*index+32)%64
}

func drawWave(screen *ebiten.Image, counter int) {
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

	op := &vector.FillOptions{
		Color: color.RGBA{0x33, 0x66, 0xff, 0xff},
	}
	path.Fill(screen, op)
}

type Game struct {
	counter int
}

func (g *Game) Update(screen *ebiten.Image) error {
	g.counter++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	drawEbitenText(screen)
	drawEbitenLogo(screen, 20, 90)
	drawWave(screen, g.counter)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.CurrentTPS(), ebiten.CurrentFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{counter: 0}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Vector (Ebiten Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

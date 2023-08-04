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

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

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
	counter int

	vertices []ebiten.Vertex
	indices  []uint16

	aa         bool
	showCenter bool
}

func (g *Game) Update() error {
	g.counter++
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.aa = !g.aa
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showCenter = !g.showCenter
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	target := screen

	joins := []vector.LineJoin{
		vector.LineJoinMiter,
		vector.LineJoinMiter,
		vector.LineJoinBevel,
		vector.LineJoinRound,
	}
	caps := []vector.LineCap{
		vector.LineCapButt,
		vector.LineCapRound,
		vector.LineCapSquare,
	}

	ow, oh := target.Bounds().Dx(), target.Bounds().Dy()
	size := min(ow/(len(joins)+1), oh/(len(caps)+1))
	offsetX, offsetY := (ow-size*len(joins))/2, (oh-size*len(caps))/2

	// Render the lines on the target.
	for j, cap := range caps {
		for i, join := range joins {
			r := image.Rect(i*size+offsetX, j*size+offsetY, (i+1)*size+offsetX, (j+1)*size+offsetY)
			miterLimit := float32(5)
			if i == 1 {
				miterLimit = 10
			}
			g.drawLine(target, r, cap, join, miterLimit)
		}
	}

	msg := fmt.Sprintf(`FPS: %0.2f, TPS: %0.2f
Press A to switch anti-aliasing.
Press C to switch to draw the center lines.`, ebiten.ActualFPS(), ebiten.ActualTPS())
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) drawLine(screen *ebiten.Image, region image.Rectangle, cap vector.LineCap, join vector.LineJoin, miterLimit float32) {
	c0x := float64(region.Min.X + region.Dx()/4)
	c0y := float64(region.Min.Y + region.Dy()/4)
	c1x := float64(region.Max.X - region.Dx()/4)
	c1y := float64(region.Max.Y - region.Dy()/4)
	r := float64(min(region.Dx(), region.Dy()) / 4)
	a0 := 2 * math.Pi * float64(g.counter) / (16 * ebiten.DefaultTPS)
	a1 := 2 * math.Pi * float64(g.counter) / (9 * ebiten.DefaultTPS)

	var path vector.Path
	sin0, cos0 := math.Sincos(a0)
	sin1, cos1 := math.Sincos(a1)
	path.MoveTo(float32(r*cos0+c0x), float32(r*sin0+c0y))
	path.LineTo(float32(-r*cos0+c0x), float32(-r*sin0+c0y))
	path.LineTo(float32(r*cos1+c1x), float32(r*sin1+c1y))
	path.LineTo(float32(-r*cos1+c1x), float32(-r*sin1+c1y))

	// Draw the main line in white.
	op := &vector.StrokeOptions{}
	op.LineCap = cap
	op.LineJoin = join
	op.MiterLimit = miterLimit
	op.Width = float32(r / 2)
	vs, is := path.AppendVerticesAndIndicesForStroke(g.vertices[:0], g.indices[:0], op)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 1
		vs[i].ColorG = 1
		vs[i].ColorB = 1
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, whiteSubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: g.aa,
	})

	// Draw the center line in red.
	if g.showCenter {
		op.Width = 1
		vs, is := path.AppendVerticesAndIndicesForStroke(g.vertices[:0], g.indices[:0], op)
		for i := range vs {
			vs[i].SrcX = 1
			vs[i].SrcY = 1
			vs[i].ColorR = 1
			vs[i].ColorG = 0
			vs[i].ColorB = 0
			vs[i].ColorA = 1
		}
		screen.DrawTriangles(vs, is, whiteSubImage, &ebiten.DrawTrianglesOptions{
			AntiAlias: g.aa,
		})
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	var g Game
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Lines (Ebitengine Demo)")
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}

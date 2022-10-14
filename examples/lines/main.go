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
	emptyImage = ebiten.NewImage(3, 3)

	// emptySubImage is an internal sub image of emptyImage.
	// Use emptySubImage at DrawTriangles instead of emptyImage in order to avoid bleeding edges.
	emptySubImage = emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	emptyImage.Fill(color.White)
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	counter int

	vertices []ebiten.Vertex
	indices  []uint16

	offscreen *ebiten.Image

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
	if g.aa {
		// Prepare the double-sized offscreen.
		// This is for anti-aliasing by a pseudo MSAA (multisample anti-aliasing).
		if g.offscreen != nil {
			sw, sh := screen.Size()
			ow, oh := g.offscreen.Size()
			if ow != sw*2 || oh != sh*2 {
				g.offscreen.Dispose()
				g.offscreen = nil
			}
		}
		if g.offscreen == nil {
			sw, sh := screen.Size()
			g.offscreen = ebiten.NewImage(sw*2, sh*2)
		}
		g.offscreen.Clear()
		target = g.offscreen
	}

	ow, oh := target.Size()
	size := min(ow/5, oh/4)
	offsetX, offsetY := (ow-size*4)/2, (oh-size*3)/2

	// Render the lines on the target.
	for j := 0; j < 3; j++ {
		for i, join := range []vector.LineJoin{
			vector.LineJoinMiter,
			vector.LineJoinMiter,
			vector.LineJoinBevel,
			vector.LineJoinRound} {
			r := image.Rect(i*size+offsetX, j*size+offsetY, (i+1)*size+offsetX, (j+1)*size+offsetY)
			miterLimit := float32(5)
			if i == 1 {
				miterLimit = 10
			}
			g.drawLine(target, r, join, miterLimit)
		}
	}

	if g.aa {
		// Render the offscreen to the screen.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(0.5, 0.5)
		op.Filter = ebiten.FilterLinear
		screen.DrawImage(g.offscreen, op)
	}

	msg := `Press A to switch anti-aliasing.
Press C to switch to draw the center lines.`
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) drawLine(screen *ebiten.Image, region image.Rectangle, join vector.LineJoin, miterLimit float32) {
	c0x := float64(region.Min.X + region.Dx()/4)
	c0y := float64(region.Min.Y + region.Dy()/4)
	c1x := float64(region.Max.X - region.Dx()/4)
	c1y := float64(region.Max.Y - region.Dy()/4)
	r := float64(min(region.Dx(), region.Dy()) / 4)
	a := 2 * math.Pi * float64(g.counter) / (10 * ebiten.DefaultTPS)

	var path vector.Path
	sin, cos := math.Sincos(a)
	path.MoveTo(float32(r*cos+c0x), float32(r*sin+c0y))
	path.LineTo(float32(-r*cos+c0x), float32(-r*sin+c0y))
	path.LineTo(float32(r*cos+c1x), float32(r*sin+c1y))
	path.LineTo(float32(-r*cos+c1x), float32(-r*sin+c1y))

	// Draw the main line in white.
	op := &vector.StrokeOptions{}
	op.LineJoin = join
	op.MiterLimit = miterLimit
	op.Width = float32(r / 2)
	vs, is := path.AppendVerticesAndIndicesForStroke(g.vertices[:0], g.indices[:0], op)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
	}
	screen.DrawTriangles(vs, is, emptySubImage, nil)

	// Draw the center line in red.
	if g.showCenter {
		op.Width = 1
		vs, is := path.AppendVerticesAndIndicesForStroke(g.vertices[:0], g.indices[:0], op)
		for i := range vs {
			vs[i].SrcX = 1
			vs[i].SrcY = 1
			vs[i].SrcX = 1
			vs[i].SrcY = 1
			vs[i].ColorR = 1
			vs[i].ColorG = 0
			vs[i].ColorB = 0
		}
		screen.DrawTriangles(vs, is, emptySubImage, nil)
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

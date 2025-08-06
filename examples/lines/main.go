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

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 640
)

type Game struct {
	debugui debugui.DebugUI

	counter int

	aa         bool
	showCenter bool
}

func (g *Game) Update() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Lines", image.Rect(10, 10, 260, 160), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
			ctx.Text(fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
			ctx.Checkbox(&g.aa, "Anti-aliasing")
			ctx.Checkbox(&g.showCenter, "Show center lines")
		})
		return nil
	}); err != nil {
		return err
	}
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
	offsetX, offsetY := (ow-size*len(joins))/2, (oh-size*len(caps))*3/4

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

	g.debugui.Draw(screen)
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
	strokeOp := &vector.StrokeOptions{}
	strokeOp.LineCap = cap
	strokeOp.LineJoin = join
	strokeOp.MiterLimit = miterLimit
	strokeOp.Width = float32(r / 2)
	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = g.aa
	vector.StrokePath(screen, &path, strokeOp, drawOp)

	// Draw the center line in red.
	if g.showCenter {
		strokeOp.Width = 1
		drawOp.ColorScale.ScaleWithColor(color.RGBA{0xff, 0, 0, 0xff})
		vector.StrokePath(screen, &path, strokeOp, drawOp)
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

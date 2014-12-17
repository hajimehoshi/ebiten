/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	"log"
	"math"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	count              int
	brushRenderTarget  ebiten.RenderTargetID
	canvasRenderTarget ebiten.RenderTargetID
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.count++
	}
	if g.brushRenderTarget.IsNil() {
		var err error
		g.brushRenderTarget, err = ebiten.NewRenderTargetID(1, 1, ebiten.FilterNearest)
		if err != nil {
			return err
		}

		gr.PushRenderTarget(g.brushRenderTarget)
		gr.Fill(0xff, 0xff, 0xff)
		gr.PopRenderTarget()
	}
	if g.canvasRenderTarget.IsNil() {
		var err error
		g.canvasRenderTarget, err = ebiten.NewRenderTargetID(screenWidth, screenHeight, ebiten.FilterNearest)
		if err != nil {
			return err
		}
		gr.PushRenderTarget(g.canvasRenderTarget)
		gr.Fill(0xff, 0xff, 0xff)
		gr.PopRenderTarget()
	}
	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		gr.PushRenderTarget(g.canvasRenderTarget)
		geo := ebiten.TranslateGeometry(float64(mx), float64(my))
		clr := ebiten.ScaleColor(color.RGBA{0xff, 0x40, 0x40, 0xff})
		theta := 2.0 * math.Pi * float64(g.count%60) / 60.0
		clr.Concat(ebiten.RotateHue(theta))
		ebiten.DrawWhole(gr.RenderTarget(g.brushRenderTarget), 1, 1, geo, clr)
		gr.PopRenderTarget()
	}

	ebiten.DrawWhole(gr.RenderTarget(g.canvasRenderTarget), screenWidth, screenHeight, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())

	ebitenutil.DebugPrint(gr, fmt.Sprintf("(%d, %d)", mx, my))
	return nil
}

func main() {
	g := new(Game)
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Paint (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

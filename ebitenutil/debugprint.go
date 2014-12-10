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

package ebitenutil

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/assets"
	"image/color"
)

type debugPrintState struct {
	textTexture            ebiten.TextureID
	debugPrintRenderTarget ebiten.RenderTargetID
	y                      int
}

var defaultDebugPrintState = new(debugPrintState)

func DebugPrint(ga ebiten.GameContext, gr ebiten.GraphicsContext, str string) {
	defaultDebugPrintState.DebugPrint(ga, gr, str)
}

func (d *debugPrintState) drawText(gr ebiten.GraphicsContext, str string, x, y int) {
	parts := []ebiten.TexturePart{}
	locationX, locationY := 0, 0
	for _, c := range str {
		if c == '\n' {
			locationX = 0
			locationY += assets.TextImageCharHeight
			continue
		}
		code := int(c)
		const xCharNum = assets.TextImageWidth / assets.TextImageCharWidth
		srcX := (code % xCharNum) * assets.TextImageCharWidth
		srcY := (code / xCharNum) * assets.TextImageCharHeight
		parts = append(parts, ebiten.TexturePart{
			LocationX: locationX,
			LocationY: locationY,
			Source:    ebiten.Rect{srcX, srcY, assets.TextImageCharWidth, assets.TextImageCharHeight},
		})
		locationX += assets.TextImageCharWidth
	}
	geo := ebiten.GeometryMatrixI()
	geo.Translate(float64(x)+1, float64(y))
	clr := ebiten.ColorMatrixI()
	// TODO: Is this color OK?
	clr.Scale(color.RGBA{0x80, 0x80, 0x80, 0xff})
	gr.Texture(d.textTexture).Draw(parts, geo, clr)
}

func (d *debugPrintState) DebugPrint(ga ebiten.GameContext, gr ebiten.GraphicsContext, str string) {
	if d.textTexture.IsNil() {
		img, err := assets.TextImage()
		if err != nil {
			panic(err)
		}
		d.textTexture, err = ga.NewTextureID(img, ebiten.FilterNearest)
		if err != nil {
			panic(err)
		}
	}
	if d.debugPrintRenderTarget.IsNil() {
		width, height := 256, 256
		var err error
		d.debugPrintRenderTarget, err = ga.NewRenderTargetID(width, height, ebiten.FilterNearest)
		if err != nil {
			panic(err)
		}
	}
	d.drawText(gr, str, 0, d.y)
}

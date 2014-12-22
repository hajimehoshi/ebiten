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
	"math"
)

type debugPrintState struct {
	textImage              *ebiten.Image
	debugPrintRenderTarget *ebiten.Image
	y                      int
}

var defaultDebugPrintState = new(debugPrintState)

func DebugPrint(r *ebiten.Image, str string) {
	defaultDebugPrintState.DebugPrint(r, str)
}

func (d *debugPrintState) drawText(rt *ebiten.Image, str string, x, y int, c color.Color) {
	parts := []ebiten.ImagePart{}
	locationX, locationY := 0, 0
	for _, c := range str {
		if c == '\n' {
			locationX = 0
			locationY += assets.TextImageCharHeight
			continue
		}
		code := int(c)
		const xCharNum = assets.TextImageWidth / assets.TextImageCharWidth
		srcX := float64(code%xCharNum) * assets.TextImageCharWidth
		srcY := float64(code/xCharNum) * assets.TextImageCharHeight
		parts = append(parts, ebiten.ImagePart{
			Dst: ebiten.Rect{float64(locationX), float64(locationY), assets.TextImageCharWidth, assets.TextImageCharHeight},
			Src: ebiten.Rect{srcX, srcY, assets.TextImageCharWidth, assets.TextImageCharHeight},
		})
		locationX += assets.TextImageCharWidth
	}
	geo := ebiten.TranslateGeometry(float64(x)+1, float64(y))
	cc := color.NRGBA64Model.Convert(c).(color.NRGBA64)
	r := float64(cc.R) / math.MaxUint16
	g := float64(cc.G) / math.MaxUint16
	b := float64(cc.B) / math.MaxUint16
	a := float64(cc.A) / math.MaxUint16
	clr := ebiten.ScaleColor(r, g, b, a)
	rt.DrawImage(d.textImage, parts, geo, clr)
}

func (d *debugPrintState) DebugPrint(r *ebiten.Image, str string) {
	if d.textImage == nil {
		img, err := assets.TextImage()
		if err != nil {
			panic(err)
		}
		d.textImage, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
		if err != nil {
			panic(err)
		}
	}
	if d.debugPrintRenderTarget == nil {
		width, height := 256, 256
		var err error
		d.debugPrintRenderTarget, err = ebiten.NewImage(width, height, ebiten.FilterNearest)
		if err != nil {
			panic(err)
		}
	}
	d.drawText(r, str, 1, d.y+1, color.NRGBA{0x00, 0x00, 0x00, 0x80})
	d.drawText(r, str, 0, d.y, color.NRGBA{0xff, 0xff, 0xff, 0xff})
}

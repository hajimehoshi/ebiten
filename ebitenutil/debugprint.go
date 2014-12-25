// Copyright 2014 Hajime Hoshi
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

package ebitenutil

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/assets"
	"image"
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
	dsts, srcs := []image.Rectangle{}, []image.Rectangle{}
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
		dst := image.Rect(locationX, locationY, locationX+assets.TextImageCharWidth, locationY+assets.TextImageCharHeight)
		src := image.Rect(srcX, srcY, srcX+assets.TextImageCharWidth, srcY+assets.TextImageCharHeight)
		dsts = append(dsts, dst)
		srcs = append(srcs, src)
		locationX += assets.TextImageCharWidth
	}
	cc := color.NRGBA64Model.Convert(c).(color.NRGBA64)
	r := float64(cc.R) / math.MaxUint16
	g := float64(cc.G) / math.MaxUint16
	b := float64(cc.B) / math.MaxUint16
	a := float64(cc.A) / math.MaxUint16
	clr := ebiten.ScaleColor(r, g, b, a)
	op := ebiten.DrawImageAt(x+1, y)
	op.DstParts = dsts
	op.SrcParts = srcs
	op.ColorMatrix = &clr
	rt.DrawImage(d.textImage, op)
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

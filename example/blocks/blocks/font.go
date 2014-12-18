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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"image/color"
)

func init() {
	texturePaths["font"] = "images/blocks/font.png"
}

const charWidth = 8
const charHeight = 8

func textWidth(str string) int {
	return charWidth * len(str)
}

func drawText(context ebiten.GraphicsContext, textures *Textures, str string, x, y, scale int, clr color.Color) {
	fontTextureId := textures.GetTexture("font")
	parts := []ebiten.TexturePart{}

	locationX, locationY := 0, 0
	for _, c := range str {
		if c == '\n' {
			locationX = 0
			locationY += charHeight
			continue
		}
		code := int(c)
		x := float64(code%16) * charWidth
		y := float64((code-32)/16) * charHeight
		parts = append(parts, ebiten.TexturePart{
			Dst: ebiten.Rect{float64(locationX), float64(locationY), charWidth, charHeight},
			Src: ebiten.Rect{x, y, charWidth, charHeight},
		})
		locationX += charWidth
	}

	geoMat := ebiten.GeometryMatrixI()
	geoMat.Concat(ebiten.ScaleGeometry(float64(scale), float64(scale)))
	geoMat.Concat(ebiten.TranslateGeometry(float64(x), float64(y)))
	clrMat := ebiten.ColorMatrixI()
	clrMat.Concat(ebiten.ScaleColor(clr))
	context.DrawTexture(fontTextureId, parts, geoMat, clrMat)
}

func drawTextWithShadow(context ebiten.GraphicsContext, textures *Textures, str string, x, y, scale int, clr color.Color) {
	drawText(context, textures, str, x+1, y+1, scale, color.RGBA{0, 0, 0, 0x80})
	drawText(context, textures, str, x, y, scale, clr)
}

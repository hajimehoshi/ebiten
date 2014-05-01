package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
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

func drawText(context graphics.Context, str string, x, y, scale int, clr color.Color) {
	fontTextureId := drawInfo.textures["font"]
	parts := []graphics.TexturePart{}

	locationX := 0
	locationY := 0
	for _, c := range str {
		if c == '\n' {
			locationX = 0
			locationY += charHeight
			continue
		}
		code := int(c)
		x := (code % 16) * charWidth
		y := ((code - 32) / 16) * charHeight
		parts = append(parts, graphics.TexturePart{
			LocationX: locationX,
			LocationY: locationY,
			Source:    graphics.Rect{x, y, charWidth, charHeight},
		})
		locationX += charWidth
	}

	geoMat := matrix.IdentityGeometry()
	geoMat.Scale(float64(scale), float64(scale))
	geoMat.Translate(float64(x), float64(y))
	clrMat := matrix.IdentityColor()
	clrMat.Scale(clr)
	context.Texture(fontTextureId).DrawParts(parts, geoMat, clrMat)
}

func drawTextWithShadow(context graphics.Context, str string, x, y, scale int, clr color.Color) {
	drawText(context, str, x+1, y+1, scale, color.RGBA{0, 0, 0, 0x80})
	drawText(context, str, x, y, scale, clr)
}

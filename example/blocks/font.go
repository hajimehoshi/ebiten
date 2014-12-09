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
		parts = append(parts, ebiten.TexturePart{
			LocationX: locationX,
			LocationY: locationY,
			Source:    ebiten.Rect{x, y, charWidth, charHeight},
		})
		locationX += charWidth
	}

	geoMat := ebiten.GeometryMatrixI()
	geoMat.Scale(float64(scale), float64(scale))
	geoMat.Translate(float64(x), float64(y))
	clrMat := ebiten.ColorMatrixI()
	clrMat.Scale(clr)
	context.Texture(fontTextureId).Draw(parts, geoMat, clrMat)
}

func drawTextWithShadow(context ebiten.GraphicsContext, textures *Textures, str string, x, y, scale int, clr color.Color) {
	drawText(context, textures, str, x+1, y+1, scale, color.RGBA{0, 0, 0, 0x80})
	drawText(context, textures, str, x, y, scale, clr)
}

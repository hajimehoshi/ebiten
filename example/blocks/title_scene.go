package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image/color"
)

type TitleScene struct {
	count int
}

func NewTitleScene() *TitleScene {
	return &TitleScene{}
}

func (s *TitleScene) Update(state GameState) {
	s.count++
}

func (s *TitleScene) Draw(context graphics.Context) {
	drawTitleBackground(context, s.count)
	drawLogo(context, "BLOCKS")
}

func drawTitleBackground(context graphics.Context, c int) {
	const textureWidth = 32
	const textureHeight = 32

	backgroundTextureId := drawInfo.textures["background"]
	parts := []graphics.TexturePart{}
	for j := -1; j < ScreenHeight/textureHeight+1; j++ {
		for i := 0; i < ScreenWidth/textureWidth+1; i++ {
			parts = append(parts, graphics.TexturePart{
				LocationX: i*textureWidth,
				LocationY: j*textureHeight,
				Source:    graphics.Rect{0, 0, textureWidth, textureHeight},
			})
		}
	}

	dx := -c % textureWidth / 2
	dy := c % textureHeight / 2
	geo := matrix.IdentityGeometry()
	geo.Translate(float64(dx), float64(dy))
	context.DrawTextureParts(backgroundTextureId, parts, geo, matrix.IdentityColor())
}

func drawLogo(context graphics.Context, str string) {
	const charWidth = 8
	const charHeight = 8
	
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
			Source: graphics.Rect{x, y, charWidth, charHeight},
		})
		locationX += charWidth
	}

	geo := matrix.IdentityGeometry()
	geo.Scale(4, 4)
	clr := matrix.IdentityColor()
	
	clr.Scale(color.RGBA{0x00, 0x00, 0x60, 0xff})
	context.DrawTextureParts(fontTextureId, parts, geo, clr)
}

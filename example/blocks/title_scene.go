package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image/color"
)

func init() {
	texturePaths["background"] = "images/blocks/background.png"
}

type TitleScene struct {
	count int
}

func NewTitleScene() *TitleScene {
	return &TitleScene{}
}

func (s *TitleScene) Update(state *GameState) {
	s.count++
	if state.Input.StateForKey(ui.KeySpace) == 1 {
		state.SceneManager.GoTo(NewGameScene())
	}
}

func (s *TitleScene) Draw(context graphics.Context, textures Textures) {
	drawTitleBackground(context, textures, s.count)
	drawLogo(context, textures, "BLOCKS")

	message := "PRESS SPACE TO START"
	x := (ScreenWidth - textWidth(message)) / 2
	y := ScreenHeight - 48
	drawTextWithShadow(context, textures, message, x, y, 1, color.RGBA{0x80, 0, 0, 0xff})
}

func drawTitleBackground(context graphics.Context, textures Textures, c int) {
	const textureWidth = 32
	const textureHeight = 32

	backgroundTextureId := textures.GetTexture("background")
	parts := []graphics.TexturePart{}
	for j := -1; j < ScreenHeight/textureHeight+1; j++ {
		for i := 0; i < ScreenWidth/textureWidth+1; i++ {
			parts = append(parts, graphics.TexturePart{
				LocationX: i * textureWidth,
				LocationY: j * textureHeight,
				Source:    graphics.Rect{0, 0, textureWidth, textureHeight},
			})
		}
	}

	dx := (-c / 4) % textureWidth
	dy := (c / 4) % textureHeight
	geo := matrix.IdentityGeometry()
	geo.Translate(float64(dx), float64(dy))
	clr := matrix.IdentityColor()
	context.Texture(backgroundTextureId).Draw(parts, geo, clr)
}

func drawLogo(context graphics.Context, textures Textures, str string) {
	scale := 4
	textWidth := textWidth(str) * scale
	x := (ScreenWidth - textWidth) / 2
	y := 32
	drawTextWithShadow(context, textures, str, x, y, scale, color.RGBA{0x00, 0x00, 0x80, 0xff})
}

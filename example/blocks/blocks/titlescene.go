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
	if state.Input.StateForKey(ebiten.KeySpace) == 1 {
		state.SceneManager.GoTo(NewGameScene())
	}
}

func (s *TitleScene) Draw(context ebiten.GraphicsContext, textures *Textures) {
	drawTitleBackground(context, textures, s.count)
	drawLogo(context, textures, "BLOCKS")

	message := "PRESS SPACE TO START"
	x := (ScreenWidth - textWidth(message)) / 2
	y := ScreenHeight - 48
	drawTextWithShadow(context, textures, message, x, y, 1, color.RGBA{0x80, 0, 0, 0xff})
}

func drawTitleBackground(context ebiten.GraphicsContext, textures *Textures, c int) {
	const textureWidth = 32
	const textureHeight = 32

	backgroundTexture := textures.GetTexture("background")
	parts := []ebiten.TexturePart{}
	for j := -1; j < ScreenHeight/textureHeight+1; j++ {
		for i := 0; i < ScreenWidth/textureWidth+1; i++ {
			parts = append(parts, ebiten.TexturePart{
				Dst: ebiten.Rect{float64(i * textureWidth), float64(j * textureHeight), textureWidth, textureHeight},
				Src: ebiten.Rect{0, 0, textureWidth, textureHeight},
			})
		}
	}

	dx := (-c / 4) % textureWidth
	dy := (c / 4) % textureHeight
	geo := ebiten.GeometryMatrixI()
	geo.Concat(ebiten.TranslateGeometry(float64(dx), float64(dy)))
	clr := ebiten.ColorMatrixI()
	context.DrawTexture(backgroundTexture, parts, geo, clr)
}

func drawLogo(context ebiten.GraphicsContext, textures *Textures, str string) {
	scale := 4
	textWidth := textWidth(str) * scale
	x := (ScreenWidth - textWidth) / 2
	y := 32
	drawTextWithShadow(context, textures, str, x, y, scale, color.RGBA{0x00, 0x00, 0x80, 0xff})
}

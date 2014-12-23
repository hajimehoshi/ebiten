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
	"image"
	"image/color"
)

func init() {
	imagePaths["background"] = "images/blocks/background.png"
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

func (s *TitleScene) Draw(r *ebiten.Image, images *Images) {
	drawTitleBackground(r, images, s.count)
	drawLogo(r, images, "BLOCKS")

	message := "PRESS SPACE TO START"
	x := (ScreenWidth - textWidth(message)) / 2
	y := ScreenHeight - 48
	drawTextWithShadow(r, images, message, x, y, 1, color.NRGBA{0x80, 0, 0, 0xff})
}

func drawTitleBackground(r *ebiten.Image, images *Images, c int) {
	const imageWidth = 32
	const imageHeight = 32

	backgroundImage := images.GetImage("background")
	dsts, srcs := []image.Rectangle{}, []image.Rectangle{}
	for j := -1; j < ScreenHeight/imageHeight+1; j++ {
		for i := 0; i < ScreenWidth/imageWidth+1; i++ {
			dstX := i * imageWidth
			dstY := j * imageHeight
			dsts = append(dsts, image.Rect(dstX, dstY, dstX+imageWidth, dstY+imageHeight))
			srcs = append(srcs, image.Rect(0, 0, imageWidth, imageHeight))
		}
	}

	dx := (-c / 4) % imageWidth
	dy := (c / 4) % imageHeight
	ebiten.DrawImageParts(r, dsts, backgroundImage, srcs, dx, dy)
}

func drawLogo(r *ebiten.Image, images *Images, str string) {
	scale := 4
	textWidth := textWidth(str) * scale
	x := (ScreenWidth - textWidth) / 2
	y := 32
	drawTextWithShadow(r, images, str, x, y, scale, color.NRGBA{0x00, 0x00, 0x80, 0xff})
}

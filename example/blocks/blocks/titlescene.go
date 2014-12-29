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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image"
	"image/color"
)

var imageBackground *ebiten.Image

func init() {
	var err error
	imageBackground, _, err = ebitenutil.NewImageFromFile("images/blocks/background.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

type TitleScene struct {
	count int
}

func NewTitleScene() *TitleScene {
	return &TitleScene{}
}

func (s *TitleScene) Update(state *GameState) error {
	s.count++
	if state.Input.StateForKey(ebiten.KeySpace) == 1 {
		state.SceneManager.GoTo(NewGameScene())
	}
	return nil
}

func (s *TitleScene) Draw(r *ebiten.Image) error {
	if err := drawTitleBackground(r, s.count); err != nil {
		return err
	}
	if err := drawLogo(r, "BLOCKS"); err != nil {
		return err
	}

	message := "PRESS SPACE TO START"
	x := (ScreenWidth - textWidth(message)) / 2
	y := ScreenHeight - 48
	return drawTextWithShadow(r, message, x, y, 1, color.NRGBA{0x80, 0, 0, 0xff})
}

func drawTitleBackground(r *ebiten.Image, c int) error {
	w, h := imageBackground.Size()
	dx := (-c / 4) % w
	dy := (c / 4) % h

	parts := []ebiten.ImagePart{}
	for j := -1; j < ScreenHeight/h+1; j++ {
		for i := 0; i < ScreenWidth/w+1; i++ {
			dstX := i*w + dx
			dstY := j*h + dy
			parts = append(parts, ebiten.ImagePart{
				Dst: image.Rect(dstX, dstY, dstX+w, dstY+h),
				Src: image.Rect(0, 0, w, h),
			})
		}
	}

	return r.DrawImage(imageBackground, &ebiten.DrawImageOptions{
		Parts: parts,
	})
}

func drawLogo(r *ebiten.Image, str string) error {
	scale := 4
	textWidth := textWidth(str) * scale
	x := (ScreenWidth - textWidth) / 2
	y := 32
	return drawTextWithShadow(r, str, x, y, scale, color.NRGBA{0x00, 0x00, 0x80, 0xff})
}

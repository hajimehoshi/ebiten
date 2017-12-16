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

// +build example

package blocks

import (
	"image/color"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/common"
)

var imageBackground *ebiten.Image

func init() {
	var err error
	imageBackground, _, err = ebitenutil.NewImageFromFile(ebitenutil.JoinStringsIntoFilePath("_resources", "images", "blocks", "background.png"), ebiten.FilterNearest)
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

func anyGamepadAbstractButtonPressed(i *Input) bool {
	for _, b := range gamepadAbstractButtons {
		if i.gamepadConfig.IsButtonPressed(0, b) {
			return true
		}
	}
	return false
}

func anyGamepadButtonPressed(i *Input) bool {
	bn := ebiten.GamepadButton(ebiten.GamepadButtonNum(0))
	for b := ebiten.GamepadButton(0); b < bn; b++ {
		if i.StateForGamepadButton(b) == 1 {
			return true
		}
	}
	return false
}

func (s *TitleScene) Update(state *GameState) error {
	s.count++
	if state.Input.StateForKey(ebiten.KeySpace) == 1 {
		state.SceneManager.GoTo(NewGameScene())
		return nil
	}
	if anyGamepadAbstractButtonPressed(state.Input) {
		state.SceneManager.GoTo(NewGameScene())
		return nil
	}
	if anyGamepadButtonPressed(state.Input) {
		state.SceneManager.GoTo(NewGamepadScene())
		return nil
	}
	return nil
}

func (s *TitleScene) Draw(r *ebiten.Image) {
	s.drawTitleBackground(r, s.count)
	drawLogo(r, "BLOCKS")

	message := "PRESS SPACE TO START"
	x := (ScreenWidth - common.ArcadeFont.TextWidth(message)) / 2
	y := ScreenHeight - 48
	common.ArcadeFont.DrawTextWithShadow(r, message, x, y, 1, color.NRGBA{0x80, 0, 0, 0xff})
}

func (s *TitleScene) drawTitleBackground(r *ebiten.Image, c int) {
	w, h := imageBackground.Size()
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < (ScreenWidth/w+1)*(ScreenHeight/h+2); i++ {
		op.GeoM.Reset()
		dx := (-c / 4) % w
		dy := (c / 4) % h
		dstX := (i%(ScreenWidth/w+1))*w + dx
		dstY := (i/(ScreenWidth/w+1)-1)*h + dy
		op.GeoM.Translate(float64(dstX), float64(dstY))
		r.DrawImage(imageBackground, op)
	}
}

func drawLogo(r *ebiten.Image, str string) {
	scale := 4
	textWidth := common.ArcadeFont.TextWidth(str) * scale
	x := (ScreenWidth - textWidth) / 2
	y := 32
	common.ArcadeFont.DrawTextWithShadow(r, str, x, y, scale, color.NRGBA{0x00, 0x00, 0x80, 0xff})
}

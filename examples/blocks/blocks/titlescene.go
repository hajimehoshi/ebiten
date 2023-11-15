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
	"bytes"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	rblocks "github.com/hajimehoshi/ebiten/v2/examples/resources/images/blocks"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var imageBackground *ebiten.Image

func init() {
	img, _, err := image.Decode(bytes.NewReader(rblocks.Background_png))
	if err != nil {
		panic(err)
	}
	imageBackground = ebiten.NewImageFromImage(img)
}

type TitleScene struct {
	count int
}

func anyGamepadVirtualButtonJustPressed(i *Input) bool {
	if !i.gamepadConfig.IsGamepadIDInitialized() {
		return false
	}

	for _, b := range virtualGamepadButtons {
		if i.gamepadConfig.IsButtonJustPressed(b) {
			return true
		}
	}
	return false
}

func (s *TitleScene) Update(state *GameState) error {
	s.count++
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		state.SceneManager.GoTo(NewGameScene())
		return nil
	}

	if anyGamepadVirtualButtonJustPressed(state.Input) {
		state.SceneManager.GoTo(NewGameScene())
		return nil
	}

	if state.Input.gamepadConfig.IsGamepadIDInitialized() {
		return nil
	}

	// If 'virtual' gamepad buttons are not set and any gamepad buttons are pressed,
	// go to the gamepad configuration scene.
	id := state.Input.GamepadIDButtonPressed()
	if id < 0 {
		return nil
	}
	state.Input.gamepadConfig.SetGamepadID(id)
	if state.Input.gamepadConfig.NeedsConfiguration() {
		g := &GamepadScene{}
		g.gamepadID = id
		state.SceneManager.GoTo(g)
	}
	return nil
}

func (s *TitleScene) Draw(r *ebiten.Image) {
	s.drawTitleBackground(r, s.count)
	drawLogo(r, "BLOCKS")

	message := "PRESS SPACE TO START"
	x := ScreenWidth / 2
	y := ScreenHeight - 48
	drawTextWithShadow(r, message, x, y, 1, color.RGBA{0x80, 0, 0, 0xff}, text.AlignCenter, text.AlignStart)
}

func (s *TitleScene) drawTitleBackground(r *ebiten.Image, c int) {
	w, h := imageBackground.Bounds().Dx(), imageBackground.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < (ScreenWidth/w+1)*(ScreenHeight/h+2); i++ {
		op.GeoM.Reset()
		dx := -(c / 4) % w
		dy := (c / 4) % h
		dstX := (i%(ScreenWidth/w+1))*w + dx
		dstY := (i/(ScreenWidth/w+1)-1)*h + dy
		op.GeoM.Translate(float64(dstX), float64(dstY))
		r.DrawImage(imageBackground, op)
	}
}

func drawLogo(r *ebiten.Image, str string) {
	const scale = 4
	x := ScreenWidth / 2
	y := 32
	drawTextWithShadow(r, str, x, y, scale, color.RGBA{0x00, 0x00, 0x80, 0xff}, text.AlignCenter, text.AlignStart)
}

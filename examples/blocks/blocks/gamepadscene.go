// Copyright 2015 Hajime Hoshi
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
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
)

type GamepadScene struct {
	gamepadID         int
	currentIndex      int
	countAfterSetting int
	buttonStates      []string
}

func (s *GamepadScene) Update(state *GameState) error {
	if s.currentIndex == 0 {
		state.Input.gamepadConfig.Reset()
		state.Input.gamepadConfig.SetGamepadID(s.gamepadID)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		state.Input.gamepadConfig.Reset()
		state.SceneManager.GoTo(&TitleScene{})
		return nil
	}

	if s.buttonStates == nil {
		s.buttonStates = make([]string, len(virtualGamepadButtons))
	}
	for i, b := range virtualGamepadButtons {
		if i < s.currentIndex {
			s.buttonStates[i] = strings.ToUpper(state.Input.gamepadConfig.ButtonName(b))
			continue
		}
		if s.currentIndex == i {
			s.buttonStates[i] = "_"
			continue
		}
		s.buttonStates[i] = ""
	}

	if 0 < s.countAfterSetting {
		s.countAfterSetting--
		if s.countAfterSetting <= 0 {
			state.SceneManager.GoTo(&TitleScene{})
		}
		return nil
	}

	b := virtualGamepadButtons[s.currentIndex]
	if state.Input.gamepadConfig.Scan(b) {
		s.currentIndex++
		if s.currentIndex == len(virtualGamepadButtons) {
			s.countAfterSetting = ebiten.MaxTPS()
		}
	}
	return nil
}

func (s *GamepadScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if s.buttonStates == nil {
		return
	}

	f := `GAMEPAD CONFIGURATION
(PRESS ESC TO CANCEL)


MOVE LEFT:    %s

MOVE RIGHT:   %s

DROP:         %s

ROTATE LEFT:  %s

ROTATE RIGHT: %s



%s`
	msg := ""
	if s.currentIndex == len(virtualGamepadButtons) {
		msg = "OK!"
	}
	str := fmt.Sprintf(f, s.buttonStates[0], s.buttonStates[1], s.buttonStates[2], s.buttonStates[3], s.buttonStates[4], msg)
	drawTextWithShadow(screen, str, 16, 16, 1, color.White)
}

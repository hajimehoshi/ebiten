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
	"github.com/hajimehoshi/ebiten/examples/common"
)

type GamepadScene struct {
	currentIndex      int
	countAfterSetting int
	buttonStates      []string
}

func NewGamepadScene() *GamepadScene {
	return &GamepadScene{}
}

func (s *GamepadScene) Update(state *GameState) error {
	if s.currentIndex == 0 {
		state.Input.gamepadConfig.Reset()
	}
	if state.Input.StateForKey(ebiten.KeyEscape) == 1 {
		state.Input.gamepadConfig.Reset()
		state.SceneManager.GoTo(NewTitleScene())
	}

	if s.buttonStates == nil {
		s.buttonStates = make([]string, len(gamepadStdButtons))
	}
	for i, b := range gamepadStdButtons {
		if i < s.currentIndex {
			s.buttonStates[i] = strings.ToUpper(state.Input.gamepadConfig.Name(b))
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
			state.SceneManager.GoTo(NewTitleScene())
		}
		return nil
	}

	b := gamepadStdButtons[s.currentIndex]
	if state.Input.gamepadConfig.Scan(0, b) {
		s.currentIndex++
		if s.currentIndex == len(gamepadStdButtons) {
			s.countAfterSetting = ebiten.FPS
		}
	}
	return nil
}

func (s *GamepadScene) Draw(screen *ebiten.Image) error {
	screen.Fill(color.Black)

	if s.buttonStates == nil {
		return nil
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
	if s.currentIndex == len(gamepadStdButtons) {
		msg = "OK!"
	}
	str := fmt.Sprintf(f, s.buttonStates[0], s.buttonStates[1], s.buttonStates[2], s.buttonStates[3], s.buttonStates[4], msg)
	if err := common.ArcadeFont.DrawTextWithShadow(screen, str, 16, 16, 1, color.White); err != nil {
		return err
	}
	return nil
}

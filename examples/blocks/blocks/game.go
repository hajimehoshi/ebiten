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
	"sync"

	"github.com/hajimehoshi/ebiten"
)

const ScreenWidth = 256
const ScreenHeight = 240

type GameState struct {
	SceneManager *SceneManager
	Input        *Input
}

type Game struct {
	once         sync.Once
	sceneManager *SceneManager
	input        Input
}

func NewGame() *Game {
	return &Game{
		sceneManager: NewSceneManager(NewTitleScene()),
	}
}

func (game *Game) Update(r *ebiten.Image) error {
	game.input.Update()
	game.sceneManager.Update(&GameState{
		SceneManager: game.sceneManager,
		Input:        &game.input,
	})
	game.sceneManager.Draw(r)
	return nil
}

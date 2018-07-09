// Copyright 2016 The Ebiten Authors
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

// +build example jsgo

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/2048/2048"
)

var (
	game *twenty48.Game
)

func update(screen *ebiten.Image) error {
	if err := game.Update(); err != nil {
		return err
	}
	if ebiten.IsDrawingSkipped() {
		return nil
	}
	game.Draw(screen)
	return nil
}

func main() {
	var err error
	game, err = twenty48.NewGame()
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, twenty48.ScreenWidth, twenty48.ScreenHeight, 1, "2048 (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

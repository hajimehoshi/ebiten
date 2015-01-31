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

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/example/platform/platform"
	"log"
)

const (
	ScreenWidth  = 256
	ScreenHeight = 240
	ScreenScale  = 2
)

var game = platform.NewGame()

func update(screen *ebiten.Image) error {
	game.Update()
	return game.Draw(screen)
}

func main() {
	if err := ebiten.Run(update, ScreenWidth, ScreenHeight, ScreenScale, "Platform (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

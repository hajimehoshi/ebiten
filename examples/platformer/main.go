// Copyright 2017 The Ebiten Authors
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

package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	// Settings
	width           = 1024
	height          = 512
	fullscreen      = false
	runinbackground = true
)

var (
	loadedSprite    *ebiten.Image
	leftSprite      *ebiten.Image
	rightSprite     *ebiten.Image
	idleSprite      *ebiten.Image
	backgroundImage *ebiten.Image
	err             error

	isFirstFrame bool = true

	backgroundOptions = &ebiten.DrawImageOptions{}
	charSpriteOptions = &ebiten.DrawImageOptions{}
)

func update(screen *ebiten.Image) error {
	// Draws Background Image
	screen.DrawImage(backgroundImage, backgroundOptions)

	// Resets
	charX := 0.0
	charY := 0.0

	if isFirstFrame == true {
		charY = charY + 380
		charX = charX + 50
		isFirstFrame = false
	}

	// Controls
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		// Selects preloaded sprite
		loadedSprite = leftSprite
		// Moves character 3px right
		charX = charX - 3.0
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		// Selects preloaded sprite
		loadedSprite = rightSprite
		// Moves character 3px left
		charX = charX + 3.0
	} else {
		loadedSprite = idleSprite
	}

	// Change gopher's position
	charSpriteOptions.GeoM.Translate(charX, charY)

	// FPS counter
	fps := fmt.Sprintf("FPS: %f", ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, fps)

	// Draws selected sprite image
	screen.DrawImage(loadedSprite, charSpriteOptions)

	return nil
}

func main() {
	// Settings for images
	backgroundOptions.GeoM.Scale(0.5, 0.5)
	backgroundOptions.GeoM.Translate(0, 0)
	charSpriteOptions.GeoM.Scale(0.5, 0.5)

	// Preload images
	// Credit goes to Renee French for the gopher pictures
	rightSprite, _, err = ebitenutil.NewImageFromFile("_resources/images/platformer/right.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	leftSprite, _, err = ebitenutil.NewImageFromFile("_resources/images/platformer/left.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	idleSprite, _, err = ebitenutil.NewImageFromFile("_resources/images/platformer/mainchar.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	// Background by CorvusSG
	backgroundImage, _, err = ebitenutil.NewImageFromFile("_resources/images/platformer/background.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}

	ebiten.SetRunnableInBackground(runinbackground)
	ebiten.SetFullscreen(fullscreen)

	// Starts the program
	ebiten.Run(update, width, height, 1, "Platformer")
}

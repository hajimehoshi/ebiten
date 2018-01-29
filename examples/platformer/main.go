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
	screenWidth  = 1024
	screenHeight = 512
)

var (
	loadedSprite    *ebiten.Image
	leftSprite      *ebiten.Image
	rightSprite     *ebiten.Image
	idleSprite      *ebiten.Image
	backgroundImage *ebiten.Image
)

func init() {
	// Preload images
	var err error
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
	backgroundImage, _, err = ebitenutil.NewImageFromFile("_resources/images/platformer/background.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

var (
	charX = 50
	charY = 380
)

func update(screen *ebiten.Image) error {
	// Controls
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		// Selects preloaded sprite
		loadedSprite = leftSprite
		// Moves character 3px right
		charX -= 3
	} else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		// Selects preloaded sprite
		loadedSprite = rightSprite
		// Moves character 3px left
		charX += 3
	} else {
		loadedSprite = idleSprite
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	// Draws Background Image
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	screen.DrawImage(backgroundImage, op)

	// Draws selected sprite image
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	op.GeoM.Translate(float64(charX), float64(charY))
	screen.DrawImage(loadedSprite, op)

	// FPS counter
	fps := fmt.Sprintf("FPS: %f", ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, fps)

	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Platformer (Ebiten Demo)"); err != nil {
		panic(err)
	}
}

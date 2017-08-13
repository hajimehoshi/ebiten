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
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var count = 0

func update(screen *ebiten.Image) error {
	count++
	count %= 240

	if ebiten.IsRunningSlowly() {
		return nil
	}

	cf := float64(count)
	ebitenutil.DrawLine(screen, 100, 100, 300, 100, color.RGBA{0xff, 0, 0xff, 0xff})
	ebitenutil.DrawLine(screen, 50, 150, 50, 350, color.RGBA{0xff, 0xff, 0, 0xff})
	ebitenutil.DrawLine(screen, 50, 100+cf, 200+cf, 250, color.RGBA{0x00, 0xff, 0xff, 0xff})
	ebitenutil.DrawRect(screen, 50+cf, 50+cf, 100+cf, 100+cf, color.RGBA{0x80, 0x80, 0x80, 0x80})
	ebitenutil.DrawRect(screen, 300-cf, 50, 120, 120, color.RGBA{0x00, 0x80, 0x00, 0x80})

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Shapes (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

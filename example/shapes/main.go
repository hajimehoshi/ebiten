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
	"image/color"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

func update(screen *ebiten.Image) error {
	for i := 0; i < 6; i++ {
		screen.DrawRect(2*i, 2*i, 100, 100, color.NRGBA{0x80, 0x80, 0xff, 0x80})
	}
	screen.FillRect(10, 10, 100, 100, color.NRGBA{0x80, 0x80, 0xff, 0x80})
	screen.FillRect(20, 20, 100, 100, color.NRGBA{0x80, 0x80, 0xff, 0x80})
	screen.DrawLine(130, 0, 140, 100, color.NRGBA{0xff, 0x80, 0x80, 0x80})
	screen.DrawLine(140, 0, 150, 100, color.NRGBA{0xff, 0x80, 0x80, 0x80})
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Shapes (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

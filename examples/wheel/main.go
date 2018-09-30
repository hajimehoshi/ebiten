// Copyright 2018 The Ebiten Authors
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
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var pointerImage *ebiten.Image

func init() {
	pointerImage, _ = ebiten.NewImage(4, 4, ebiten.FilterDefault)
	pointerImage.Fill(color.RGBA{0xff, 0, 0, 0xff})
}

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	x = 0.0
	y = 0.0
)

func update(screen *ebiten.Image) error {
	dx, dy := ebiten.Wheel()
	x += dx
	y += dy

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	screen.DrawImage(pointerImage, op)

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf("Move the red point by mouse wheel\n(%0.2f, %0.2f)", x, y))

	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Wheel (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

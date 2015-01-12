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
	"image"
	"image/color"
	"log"
	"math/rand"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var rectsToDraw = make([]image.Rectangle, 100)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func init() {
	for i, _ := range rectsToDraw {
		x0, x1 := rand.Intn(screenWidth), rand.Intn(screenWidth)
		y0, y1 := rand.Intn(screenHeight), rand.Intn(screenHeight)
		rectsToDraw[i] = image.Rect(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1))
	}
}

type rects []image.Rectangle

func (r rects) Len() int {
	return len(r)
}

func (r rects) Points(i int) (x0, y0, x1, y1 int) {
	rect := &r[i]
	return rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y
}

func update(screen *ebiten.Image) error {
	screen.DrawRects(color.NRGBA{0x80, 0x80, 0xff, 0x80}, rects(rectsToDraw))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Shapes (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

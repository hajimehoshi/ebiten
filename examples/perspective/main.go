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

package main

import (
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	gophersImage *ebiten.Image
)

type parts struct {
	image *ebiten.Image
}

func (p parts) Len() int {
	_, h := p.image.Size()
	return h
}

func (p parts) Dst(i int) (x0, y0, x1, y1 int) {
	w, h := p.image.Size()
	width := w + i*3/4
	x := ((h - i) * 3 / 4) / 2
	return x, i, x + width, i + 1
}

func (p parts) Src(i int) (x0, y0, x1, y1 int) {
	w, _ := p.image.Size()
	return 0, i, w, i + 1
}

func update(screen *ebiten.Image) error {
	op := &ebiten.DrawImageOptions{
		ImageParts: &parts{gophersImage},
	}
	w, h := gophersImage.Size()
	maxWidth := float64(w) + float64(h)*0.75
	op.GeoM.Translate(-maxWidth/2, -float64(h)/2)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	screen.DrawImage(gophersImage, op)
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

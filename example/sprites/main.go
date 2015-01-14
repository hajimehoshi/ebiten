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
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	_ "image/png"
	"log"
	"math/rand"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	ebitenImage *ebiten.Image
)

type Sprite struct {
	image *ebiten.Image
	x     int
	y     int
	vx    int
	vy    int
}

func (s *Sprite) Update() {
	s.x += s.vx
	s.y += s.vy
	if s.x < 0 {
		s.x = -s.x
		s.vx = -s.vx
	}
	if s.y < 0 {
		s.y = -s.y
		s.vy = -s.vy
	}
	w, h := s.image.Size()
	if screenWidth <= s.x+w {
		s.x = 2*(screenWidth-w) - s.x
		s.vx = -s.vx
	}
	if screenHeight <= s.y+h {
		s.y = 2*(screenHeight-h) - s.y
		s.vy = -s.vy
	}
}

func (s *Sprite) Dst() (x0, y0, x1, y1 int) {
	w, h := s.image.Size()
	return s.x, s.y, s.x + w, s.y + h
}

func (s *Sprite) Src() (x0, y0, x1, y1 int) {
	w, h := s.image.Size()
	return 0, 0, w, h
}

type Sprites []*Sprite

func (s Sprites) Update() {
	for _, sprite := range s {
		sprite.Update()
	}
}

func (s Sprites) Len() int {
	return len(s)
}

func (s Sprites) Dst(i int) (x0, y0, x1, y1 int) {
	return s[i].Dst()
}

func (s Sprites) Src(i int) (x0, y0, x1, y1 int) {
	return s[i].Src()
}

var sprites = make(Sprites, 10000)

func update(screen *ebiten.Image) error {
	sprites.Update()
	op := &ebiten.DrawImageOptions{
		ImageParts: sprites,
	}
	op.ColorM.Scale(1.0, 1.0, 1.0, 0.5)
	if err := screen.DrawImage(ebitenImage, op); err != nil {
		return err
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f\nNum of sprites: %d", ebiten.CurrentFPS(), sprites.Len()))
	return nil
}

func main() {
	var err error
	ebitenImage, _, err = ebitenutil.NewImageFromFile("images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	for i, _ := range sprites {
		w, h := ebitenImage.Size()
		x, y := rand.Intn(screenWidth-w), rand.Intn(screenHeight-h)
		vx, vy := 2*rand.Intn(2)-1, 2*rand.Intn(2)-1
		sprites[i] = &Sprite{
			image: ebitenImage,
			x:     x,
			y:     y,
			vx:    vx,
			vy:    vy,
		}
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Sprites (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

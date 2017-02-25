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

// +build example

package main

import (
	"fmt"
	_ "image/png"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	ebitenImage       *ebiten.Image
	ebitenImageWidth  = 0
	ebitenImageHeight = 0
)

type Sprite struct {
	imageWidth  int
	imageHeight int
	x           int
	y           int
	vx          int
	vy          int
}

func (s *Sprite) Update() {
	s.x += s.vx
	s.y += s.vy
	if s.x < 0 {
		s.x = -s.x
		s.vx = -s.vx
	} else if screenWidth <= s.x+s.imageWidth {
		s.x = 2*(screenWidth-s.imageWidth) - s.x
		s.vx = -s.vx
	}
	if s.y < 0 {
		s.y = -s.y
		s.vy = -s.vy
	} else if screenHeight <= s.y+s.imageHeight {
		s.y = 2*(screenHeight-s.imageHeight) - s.y
		s.vy = -s.vy
	}
}

type Sprites struct {
	sprites []*Sprite
	num     int
}

func (s *Sprites) Update() {
	for _, sprite := range s.sprites {
		sprite.Update()
	}
}

func (s *Sprites) Len() int {
	return s.num
}

func (s *Sprites) Dst(i int) (x0, y0, x1, y1 int) {
	if s.num <= i {
		return 0, 0, 0, 0
	}
	ss := s.sprites[i]
	return ss.x, ss.y, ss.x + ebitenImageWidth, ss.y + ebitenImageHeight
}

func (s *Sprites) Src(i int) (x0, y0, x1, y1 int) {
	if s.num <= i {
		return 0, 0, 0, 0
	}
	return 0, 0, ebitenImageWidth, ebitenImageHeight
}

const (
	MinSprites = 0
	MaxSprites = 50000
)

var (
	sprites *Sprites
	op      *ebiten.DrawImageOptions
)

func update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		sprites.num -= 20
		if sprites.num < MinSprites {
			sprites.num = MinSprites
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		sprites.num += 20
		if MaxSprites < sprites.num {
			sprites.num = MaxSprites
		}
	}
	sprites.Update()

	if ebiten.IsRunningSlowly() {
		return nil
	}
	if err := screen.DrawImage(ebitenImage, op); err != nil {
		return err
	}
	msg := fmt.Sprintf(`FPS: %0.2f
Num of sprites: %d
Press <- or -> to change the number of sprites`, ebiten.CurrentFPS(), sprites.Len())
	if err := ebitenutil.DebugPrint(screen, msg); err != nil {
		return err
	}
	return nil
}

func main() {
	sprites = &Sprites{make([]*Sprite, MaxSprites), 500}
	op = &ebiten.DrawImageOptions{
		ImageParts: sprites,
	}
	op.ColorM.Scale(1.0, 1.0, 1.0, 0.5)
	var err error
	ebitenImage, _, err = ebitenutil.NewImageFromFile("_resources/images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	ebitenImageWidth, ebitenImageHeight = ebitenImage.Size()
	for i := range sprites.sprites {
		w, h := ebitenImage.Size()
		x, y := rand.Intn(screenWidth-w), rand.Intn(screenHeight-h)
		vx, vy := 2*rand.Intn(2)-1, 2*rand.Intn(2)-1
		sprites.sprites[i] = &Sprite{
			imageWidth:  w,
			imageHeight: h,
			x:           x,
			y:           y,
			vx:          vx,
			vy:          vy,
		}
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Sprites (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

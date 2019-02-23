// Copyright 2019 The Ebiten Authors
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
	"bytes"
	"container/list"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	screenWidth  = 640
	screenHeight = 480
)

var smokeImage *ebiten.Image

func init() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Smoke_png))
	if err != nil {
		log.Fatal(err)
	}
	smokeImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

type sprite struct {
	count    int
	maxCount int
	angle    float64

	img   *ebiten.Image
	scale float64
	alpha float64
}

func (s *sprite) update() {
	if s.count == 0 {
		return
	}
	s.count--
}

func (s *sprite) terminated() bool {
	return s.count == 0
}

func (s *sprite) draw(screen *ebiten.Image) {
	if s.count == 0 {
		return
	}

	const (
		ox = screenWidth / 2
		oy = screenHeight / 2
	)
	x := math.Cos(s.angle) * float64(s.maxCount-s.count)
	y := math.Sin(s.angle) * float64(s.maxCount-s.count)

	op := &ebiten.DrawImageOptions{}

	sx, sy := s.img.Size()
	op.GeoM.Translate(-float64(sx)/2, -float64(sy)/2)
	op.GeoM.Scale(s.scale, s.scale)
	op.GeoM.Translate(x, y)
	op.GeoM.Translate(ox, oy)

	rate := float64(s.count) / float64(s.maxCount)
	alpha := 0.0
	if rate < 0.2 {
		alpha = rate / 0.2
	} else if rate > 0.8 {
		alpha = (1 - rate) / 0.2
	} else {
		alpha = 1
	}
	alpha *= s.alpha
	op.ColorM.Scale(1, 1, 1, alpha)

	op.Filter = ebiten.FilterLinear

	screen.DrawImage(s.img, op)
}

func newSprite(img *ebiten.Image) *sprite {
	c := rand.Intn(50) + 300
	a := rand.Float64() * 2 * math.Pi
	s := rand.Float64()*0.1 + 0.4
	return &sprite{
		img: img,

		maxCount: c,
		count:    c,
		angle:    a,
		scale:    s,
		alpha:    0.5,
	}
}

var sprites = list.New()

func update(screen *ebiten.Image) error {
	if sprites.Len() < 500 {
		// Emit
		sprites.PushBack(newSprite(smokeImage))
	}

	for e := sprites.Front(); e != nil; e = e.Next() {
		s := e.Value.(*sprite)
		s.update()
		if s.terminated() {
			defer sprites.Remove(e)
		}
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	screen.Fill(color.RGBA{0x99, 0xcc, 0xff, 0xff})
	for e := sprites.Front(); e != nil; e = e.Next() {
		s := e.Value.(*sprite)
		s.draw(screen)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nSprites: %d", ebiten.CurrentTPS(), sprites.Len()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Particles (Ebiten demo)"); err != nil {
		log.Fatal(err)
	}
}

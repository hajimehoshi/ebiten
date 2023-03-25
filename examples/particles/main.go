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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
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
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Smoke_png))
	if err != nil {
		log.Fatal(err)
	}
	smokeImage = ebiten.NewImageFromImage(img)
}

type sprite struct {
	count    int
	maxCount int
	dir      float64

	img   *ebiten.Image
	scale float64
	angle float64
	alpha float32
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
	x := math.Cos(s.dir) * float64(s.maxCount-s.count)
	y := math.Sin(s.dir) * float64(s.maxCount-s.count)

	op := &ebiten.DrawImageOptions{}

	sx, sy := s.img.Bounds().Dx(), s.img.Bounds().Dy()
	op.GeoM.Translate(-float64(sx)/2, -float64(sy)/2)
	op.GeoM.Rotate(s.angle)
	op.GeoM.Scale(s.scale, s.scale)
	op.GeoM.Translate(x, y)
	op.GeoM.Translate(ox, oy)

	rate := float32(s.count) / float32(s.maxCount)
	var alpha float32
	if rate < 0.2 {
		alpha = rate / 0.2
	} else if rate > 0.8 {
		alpha = (1 - rate) / 0.2
	} else {
		alpha = 1
	}
	alpha *= s.alpha
	op.ColorScale.ScaleAlpha(alpha)

	screen.DrawImage(s.img, op)
}

func newSprite(img *ebiten.Image) *sprite {
	c := rand.Intn(50) + 300
	dir := rand.Float64() * 2 * math.Pi
	a := rand.Float64() * 2 * math.Pi
	s := rand.Float64()*0.1 + 0.4
	return &sprite{
		img: img,

		maxCount: c,
		count:    c,
		dir:      dir,

		angle: a,
		scale: s,
		alpha: 0.5,
	}
}

type Game struct {
	sprites *list.List
}

func (g *Game) Update() error {
	if g.sprites == nil {
		g.sprites = list.New()
	}

	if g.sprites.Len() < 500 && rand.Intn(4) < 3 {
		// Emit
		g.sprites.PushBack(newSprite(smokeImage))
	}

	for e := g.sprites.Front(); e != nil; e = e.Next() {
		s := e.Value.(*sprite)
		s.update()
		if s.terminated() {
			defer g.sprites.Remove(e)
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x99, 0xcc, 0xff, 0xff})
	for e := g.sprites.Front(); e != nil; e = e.Next() {
		s := e.Value.(*sprite)
		s.draw(screen)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nSprites: %d", ebiten.ActualTPS(), g.sprites.Len()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Particles (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

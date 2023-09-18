// Copyright 2020 The Ebiten Authors
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

// Mascot is a desktop mascot on cross platforms.
// This is inspired by mattn's gopher (https://github.com/mattn/gopher).
package main

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	rmascot "github.com/hajimehoshi/ebiten/v2/examples/resources/images/mascot"
)

const (
	width  = 200
	height = 200
)

var (
	gopher1 *ebiten.Image
	gopher2 *ebiten.Image
	gopher3 *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img1, _, err := image.Decode(bytes.NewReader(rmascot.Out01_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher1 = ebiten.NewImageFromImage(img1)

	img2, _, err := image.Decode(bytes.NewReader(rmascot.Out02_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher2 = ebiten.NewImageFromImage(img2)

	img3, _, err := image.Decode(bytes.NewReader(rmascot.Out03_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher3 = ebiten.NewImageFromImage(img3)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type mascot struct {
	x16  int
	y16  int
	vx16 int
	vy16 int

	count int
}

func (m *mascot) Layout(outsideWidth, outsideHeight int) (int, int) {
	return width, height
}

func (m *mascot) Update() error {
	m.count++

	sw, sh := ebiten.ScreenSizeInFullscreen()
	ebiten.SetWindowPosition(m.x16/16, m.y16/16+sh-height)

	if m.vx16 == 0 {
		m.vx16 = 64
	}
	m.x16 += m.vx16
	if m.x16/16 > sw-width && m.vx16 > 0 {
		m.vx16 = -64
	}
	if m.x16 <= 0 && m.vx16 < 0 {
		m.vx16 = 64
	}

	// Accelerate the mascot in the Y direction.
	m.vy16 += 8
	m.y16 += m.vy16

	// If the mascot is on the ground, stop it in the Y direction.
	if m.y16 >= 0 {
		m.y16 = 0
		m.vy16 = 0
	}

	// If the mascto is on the ground, cause an action in random.
	if rand.Intn(60) == 0 && m.y16 == 0 {
		switch rand.Intn(2) {
		case 0:
			// Jump.
			m.vy16 = -240
		case 1:
			// Turn.
			m.vx16 = -m.vx16
		}
	}
	return nil
}

func (m *mascot) Draw(screen *ebiten.Image) {
	img := gopher1
	if m.y16 == 0 {
		switch (m.count / 3) % 4 {
		case 0:
			img = gopher1
		case 1, 3:
			img = gopher2
		case 2:
			img = gopher3
		}
	}
	op := &ebiten.DrawImageOptions{}
	if m.vx16 < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(width, 0)
	}
	screen.DrawImage(img, op)
}

func main() {
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowMousePassthrough(true)

	op := &ebiten.RunGameOptions{}
	op.ScreenTransparent = true
	op.SkipTaskbar = true
	if err := ebiten.RunGameWithOptions(&mascot{}, op); err != nil {
		log.Fatal(err)
	}
}

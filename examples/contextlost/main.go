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

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	gophersImage *ebiten.Image
	extraImages  []*ebiten.Image
)

type Game struct {
	count int
	lost  bool
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.loseAndRestoreContext()
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		ebiten.SetScreenClearedEveryFrame(!ebiten.IsScreenClearedEveryFrame())
	}

	g.count++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.lost {
		// When the context is lost, skip rendering. Usually this logic should not be required, but when the
		// context lost happens by the API explicitly, Draw can be called even after the data in GPU
		// disappeared.
		return
	}

	s := gophersImage.Bounds().Size()
	op := &ebiten.DrawImageOptions{}

	// For the details, see examples/rotate.
	op.GeoM.Translate(-float64(s.X)/2, -float64(s.Y)/2)
	op.GeoM.Rotate(float64(g.count%360) * 2 * math.Pi / 360)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	screen.DrawImage(gophersImage, op)

	msg := `Press Space to force to lose/restore the GL context!
(Browser only)

Press S to switch clearing the screen
at the beginning of each frame.`
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	// Extend the shared backend GL texture on purpose.
	for i := 0; i < 20; i++ {
		eimg := ebiten.NewImageFromImage(img)
		extraImages = append(extraImages, eimg)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Context Lost (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

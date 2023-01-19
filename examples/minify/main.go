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

// This example is an experiment to minify images with various filters.
// When linear filter is used, mipmap images should be used for high-quality rendering (#578).

package main

import (
	"bytes"
	"fmt"
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
	screenWidth  = 800
	screenHeight = 480
)

var (
	gophersImage *ebiten.Image
)

type Game struct {
	rotate  bool
	clip    bool
	counter int
}

func (g *Game) Update() error {
	g.counter++
	if g.counter == 480 {
		g.counter = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.rotate = !g.rotate
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.clip = !g.clip
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	s := 1.5 / math.Pow(1.01, float64(g.counter))

	clippedGophersImage := gophersImage.SubImage(image.Rect(100, 100, 200, 200)).(*ebiten.Image)
	for i, f := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()

		op := &ebiten.DrawImageOptions{}
		if g.rotate {
			op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			op.GeoM.Rotate(float64(g.counter) / 300 * 2 * math.Pi)
			op.GeoM.Translate(float64(w)/2, float64(h)/2)
		}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(32+float64(i*w)*s+float64(i*4), 64)
		op.Filter = f
		if g.clip {
			screen.DrawImage(clippedGophersImage, op)
		} else {
			screen.DrawImage(gophersImage, op)
		}
	}

	msg := fmt.Sprintf(`Minifying images (Nearest filter vs Linear filter):
Press R to rotate the images.
Press C to clip the images.
Scale: %0.2f`, s)
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

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Minify (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

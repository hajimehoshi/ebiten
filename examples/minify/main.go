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

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
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

func (g *Game) Update(screen *ebiten.Image) error {
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
	msg := fmt.Sprintf(`Minifying images (Nearest filter vs Linear filter):
Press R to rotate the images.
Press C to clip the images.
Scale: %0.2f`, s)
	ebitenutil.DebugPrint(screen, msg)

	clippedGophersImage := gophersImage.SubImage(image.Rect(100, 100, 200, 200)).(*ebiten.Image)
	for i, f := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		w, h := gophersImage.Size()

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
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}

	// Specifying filter on NewImage[FromImage] is just for backward compatibility.
	// Now specifying filter at DrawImageOptions is recommended.
	// Specify FilterDefault here, that means to prefer filter specified at DrawImageOptions.
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Minify (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

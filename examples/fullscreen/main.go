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

// +build example

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/text"
)

var (
	gophersImage       *ebiten.Image
	mplusFont          font.Face
	regularTermination = errors.New("regular termination")
)

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
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)
}

func initFont() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12 * ebiten.DeviceScaleFactor(),
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return regularTermination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	scale := ebiten.DeviceScaleFactor()

	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(float64(g.count%360) * 2 * math.Pi / 360)
	sw, sh := screen.Size()
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(gophersImage, op)

	fw, fh := ebiten.ScreenSizeInFullscreen()
	msgs := []string{
		"This is an example of the finest fullscreen. Press Q to quit.",
		fmt.Sprintf("Screen size in fullscreen: %d, %d", fw, fh),
		fmt.Sprintf("Game's screen size: %d, %d", sw, sh),
		fmt.Sprintf("Device scale factor: %0.2f", scale),
	}

	for i, msg := range msgs {
		text.Draw(screen, msg, mplusFont, int(50*scale), int(50+float64(i)*16*scale), color.White)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
	// Call initFont here instead of init funcs since ebiten.DeviceScaleFactor is not available in init.
	initFont()

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Fullscreen (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil && err != regularTermination {
		log.Fatal(err)
	}
}

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
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"
	"runtime"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

var (
	gophersImage *ebiten.Image
	mplusFont    font.Face
)

func init() {
	// Decode an image from the image file's byte slice.
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
		Hinting: font.HintingVertical,
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

	if runtime.GOOS == "js" {
		if ebiten.IsKeyPressed(ebiten.KeyF) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			ebiten.SetFullscreen(true)
		}
	}
	if runtime.GOOS != "js" && ebiten.IsKeyPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	scale := ebiten.DeviceScaleFactor()

	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(float64(g.count%360) * 2 * math.Pi / 360)
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(gophersImage, op)

	fw, fh := ebiten.ScreenSizeInFullscreen()
	msg := "This is an example of the finest fullscreen.\n"
	if runtime.GOOS == "js" {
		msg += "Press F or touch the screen to enter fullscreen (again).\n"
	} else {
		msg += "Press Q to quit.\n"
	}
	msg += fmt.Sprintf("Screen size in fullscreen: %d, %d\n", fw, fh)
	msg += fmt.Sprintf("Game's screen size: %d, %d\n", sw, sh)
	msg += fmt.Sprintf("Device scale factor: %0.2f\n", scale)

	text.Draw(screen, msg, mplusFont, int(50*scale), 50, color.White)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
	// Call initFont here instead of init funcs since ebiten.DeviceScaleFactor is not available in init.
	initFont()

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Fullscreen (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

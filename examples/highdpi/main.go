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
	"fmt"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	highDPIImageCh chan *ebiten.Image
	highDPIImage   *ebiten.Image
}

func NewGame() *Game {
	g := &Game{
		highDPIImageCh: make(chan *ebiten.Image),
	}

	// Licensed under Public Domain
	// https://commons.wikimedia.org/wiki/File:As08-16-2593.jpg
	const url = "https://upload.wikimedia.org/wikipedia/commons/1/1f/As08-16-2593.jpg"

	// Load the image asynchronously.
	go func() {
		img, err := ebitenutil.NewImageFromURL(url)
		if err != nil {
			log.Fatal(err)
		}
		g.highDPIImageCh <- img
		close(g.highDPIImageCh)
	}()

	return g
}

func (g *Game) Update() error {
	if g.highDPIImage != nil {
		return nil
	}

	// Use select and 'default' clause for non-blocking receiving.
	select {
	case img := <-g.highDPIImageCh:
		g.highDPIImage = img
	default:
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.highDPIImage == nil {
		ebitenutil.DebugPrint(screen, "Loading the image...")
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	w, h := g.highDPIImage.Bounds().Dx(), g.highDPIImage.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}

	// Move the images's center to the upper left corner.
	op.GeoM.Translate(float64(-w)/2, float64(-h)/2)

	// The image is just too big. Adjust the scale.
	op.GeoM.Scale(0.25, 0.25)

	// Scale the image by the device ratio so that the rendering result can be same
	// on various (different-DPI) environments.
	scale := ebiten.DeviceScaleFactor()
	op.GeoM.Scale(scale, scale)

	// Move the image's center to the screen's center.
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)

	op.Filter = ebiten.FilterLinear
	screen.DrawImage(g.highDPIImage, op)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("(Init) Device Scale Ratio: %0.2f", scale))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// The unit of outsideWidth/Height is device-independent pixels.
	// By multiplying them by the device scale factor, we can get a hi-DPI screen size.
	s := ebiten.DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("High DPI (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

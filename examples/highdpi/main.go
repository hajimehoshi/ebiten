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

package main

import (
	"fmt"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var (
	count          int
	highDPIImage   *ebiten.Image
	highDPIImageCh = make(chan *ebiten.Image)

	// scale represents the device scale when the application starts.
	scale = 1.0
)

func init() {
	// Licensed under Public Domain
	// https://commons.wikimedia.org/wiki/File:As08-16-2593.jpg
	const url = "https://upload.wikimedia.org/wikipedia/commons/1/1f/As08-16-2593.jpg"

	// Load the image asynchronously.
	go func() {
		img, err := ebitenutil.NewImageFromURL(url)
		if err != nil {
			log.Fatal(err)
		}
		highDPIImageCh <- img
		close(highDPIImageCh)
	}()
}

func update(screen *ebiten.Image) error {
	// TODO: DeviceScaleFactor() might return different values for different monitors.
	// Add a mode to adjust the screen size along with the current device scale (#705).
	// Now this example uses the device scale initialized at the begining of this application.

	if highDPIImage == nil {
		// Use select and 'default' clause for non-blocking receiving.
		select {
		case img := <-highDPIImageCh:
			highDPIImage = img
		default:
		}
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	if highDPIImage == nil {
		ebitenutil.DebugPrint(screen, "Loading the image...")
		return nil
	}

	sw, sh := screen.Size()

	w, h := highDPIImage.Size()
	op := &ebiten.DrawImageOptions{}

	// Move the images's center to the upper left corner.
	op.GeoM.Translate(float64(-w)/2, float64(-h)/2)

	// The image is just too big. Adjust the scale.
	op.GeoM.Scale(0.25, 0.25)

	// Scale the image by the device ratio so that the rendering result can be same
	// on various (diffrent-DPI) environments.
	op.GeoM.Scale(scale, scale)

	// Move the image's center to the screen's center.
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)

	op.Filter = ebiten.FilterLinear
	screen.DrawImage(highDPIImage, op)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("(Init) Device Scale Ratio: %0.2f", scale))
	return nil
}

func main() {
	const (
		screenWidth  = 640
		screenHeight = 480
	)

	// Pass the invert of scale so that Ebiten's auto scaling by device scale is disabled.
	//
	// Note that DeviceScaleFactor cannot be called from init() functions on some environments.
	scale = ebiten.DeviceScaleFactor()
	s := scale
	if err := ebiten.Run(update, int(screenWidth*s), int(screenHeight*s), 1/s, "High DPI (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

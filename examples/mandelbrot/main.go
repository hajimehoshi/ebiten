// Copyright 2017 The Ebiten Authors
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
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
)

const (
	screenWidth  = 640
	screenHeight = 640
	maxIt        = 128
)

var (
	offscreen    *ebiten.Image
	offscreenPix []byte
	palette      [maxIt]byte
)

func init() {
	offscreen, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	offscreenPix = make([]byte, screenWidth*screenHeight*4)
	for i := range palette {
		c := byte(math.Sqrt(float64(i)/float64(len(palette))) * 0xff)
		palette[i] = c
	}
}

func color(it int) (r, g, b byte) {
	if it == maxIt {
		return 0xff, 0xff, 0xff
	}
	c := palette[it]
	return c, c, c
}

func updateOffscreen(centerX, centerY, size float64) {
	for j := 0; j < screenHeight; j++ {
		for i := 0; i < screenHeight; i++ {
			x := float64(i)*size/screenWidth - size/2 + centerX
			y := (screenHeight-float64(j))*size/screenHeight - size/2 + centerY
			c := complex(x, y)
			z := c
			it := 0
			for ; it < maxIt; it++ {
				nz := z*z + c
				if real(nz)*real(nz)+imag(nz)*imag(nz) > 4 {
					break
				}
				z = nz
			}
			r, g, b := color(it)
			p := 4 * (i + j*screenWidth)
			offscreenPix[p] = r
			offscreenPix[p+1] = g
			offscreenPix[p+2] = b
			offscreenPix[p+3] = 0xff
		}
	}
	offscreen.ReplacePixels(offscreenPix)
}

func init() {
	// Now it is not feasible to call updateOffscreen every frame due to performance.
	updateOffscreen(-0.75, 0.25, 2)
}

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.DrawImage(offscreen, nil)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Mandelbrot (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

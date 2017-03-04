// Copyright 2015 Hajime Hoshi
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
	"fmt"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	noiseImage *image.RGBA
)

type rand struct {
	x, y, z, w uint32
}

func (r *rand) next() uint32 {
	// math/rand is too slow to keep 60 FPS on web browsers.
	// Use Xorshift instead: http://en.wikipedia.org/wiki/Xorshift
	t := r.x ^ (r.x << 11)
	r.x, r.y, r.z = r.y, r.z, r.w
	r.w = (r.w ^ (r.w >> 19)) ^ (t ^ (t >> 8))
	return r.w
}

var randInstance = &rand{12345678, 4185243, 776511, 45411}

func update(screen *ebiten.Image) error {
	const l = screenWidth * screenHeight
	for i := 0; i < l; i++ {
		x := randInstance.next()
		noiseImage.Pix[4*i] = uint8(x >> 24)
		noiseImage.Pix[4*i+1] = uint8(x >> 16)
		noiseImage.Pix[4*i+2] = uint8(x >> 8)
		noiseImage.Pix[4*i+3] = 0xff
	}
	screen.ReplacePixels(noiseImage.Pix)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	noiseImage = image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Noise (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

// Copyright 2022 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	images       []*ebiten.Image
	imageCreated chan struct{}
}

func (g *Game) Update() error {
	if g.imageCreated == nil {
		g.imageCreated = make(chan struct{})
		go func() {
			op := &ebiten.NewImageFromImageOptions{
				Unmanaged: true,
			}
			for i := 0; i < 3; i++ {
				i := i
				img := image.NewRGBA(image.Rect(0, 0, 512, 512))
				for j := 0; j < len(img.Pix)/4; j++ {
					img.Pix[4*j] = byte(0x60 * i)
					img.Pix[4*j+1] = byte(0x60 * i)
					img.Pix[4*j+2] = byte(0x60 * i)
					img.Pix[4*j+3] = 0xff
				}
				g.images = append(g.images, ebiten.NewImageFromImageWithOptions(img, op))
			}
			close(g.imageCreated)
		}()
		return nil
	}

	select {
	case <-g.imageCreated:
	default:
		return nil
	}

	for i, img := range g.images {
		got := img.At(0, 0).(color.RGBA)
		want := color.RGBA{byte(0x60 * i), byte(0x60 * i), byte(0x60 * i), 0xff}
		if got != want {
			panic(fmt.Sprintf("got: %v, want: %v", got, want))
		}
	}

	return ebiten.Termination
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

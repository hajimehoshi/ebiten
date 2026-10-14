// Copyright 2026 The Ebitengine Authors
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
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	drew bool
}

func checkReadback() {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.RGBA{G: 123, A: 255})
	pixels := make([]byte, 4)
	result := img.ReadPixelsAsync(pixels)
	img.Fill(color.White)
	img.Deallocate()
	if err := result.Wait(); err != nil {
		panic(err)
	}
	if pixels[0] != 0 || pixels[1] != 123 || pixels[2] != 0 || pixels[3] != 255 {
		panic(fmt.Sprintf("unexpected pixels: %v", pixels))
	}
}

func (g *game) Update() error {
	checkReadback()
	if g.drew {
		return ebiten.Termination
	}
	return nil
}

func (g *game) Draw(*ebiten.Image) {
	checkReadback()
	g.drew = true
}

func (*game) Layout(int, int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowVisible(false)
	if err := ebiten.RunGame(&game{}); err != nil {
		panic(err)
	}
}

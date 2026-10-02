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
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"time"
)

type game struct {
	result <-chan error
	pixels []byte
	image  *ebiten.Image
}

func (g *game) Update() error {
	if g.result == nil {
		g.image = ebiten.NewImage(16, 16)
		sub := g.image.SubImage(image.Rect(3, 4, 5, 6)).(*ebiten.Image)
		sub.Fill(color.RGBA{R: 123, A: 255})
		g.pixels = make([]byte, 16)
		g.result = sub.ReadPixelsAsync(g.pixels)
		sub.Fill(color.White)
		g.image.Dispose()
		return nil
	}
	select {
	case err := <-g.result:
		if err != nil {
			return err
		}
		for n, v := range g.pixels {
			if want := []byte{123, 0, 0, 255}[n%4]; v != want {
				return fmt.Errorf("pixel %d: got %d, want %d", n, v, want)
			}
		}
		// This request has no opportunity to reach the graphics driver before exit.
		img := ebiten.NewImage(1, 1)
		img.Fill(color.White)
		g.result = img.ReadPixelsAsync(make([]byte, 4))
		return ebiten.Termination
	default:
		return nil
	}
}
func (*game) Draw(*ebiten.Image)         {}
func (*game) Layout(int, int) (int, int) { return 16, 16 }
func main() {
	ebiten.SetWindowVisible(false)
	g := &game{}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	select {
	case err := <-g.result:
		if err == nil {
			panic("unsubmitted read-back succeeded")
		}
	case <-time.After(5 * time.Second):
		panic("unsubmitted read-back was not aborted on exit")
	}
}

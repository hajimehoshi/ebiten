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
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	result *ebiten.PixelReadback
	pixels []byte
	image  *ebiten.Image
	reads  int
}

func (g *game) Update() error {
	if g.result == nil {
		g.image = ebiten.NewImage(16, 16)
		width := 2 + (g.reads/3)%2
		sub := g.image.SubImage(image.Rect(3, 4, 3+width, 6)).(*ebiten.Image)
		sub.Fill(color.RGBA{R: byte(123 + g.reads), A: 255})
		g.pixels = make([]byte, 4*width*2)
		g.result = sub.ReadPixelsAsync(g.pixels)
		sub.Fill(color.White)
		g.image.Deallocate()
		return nil
	}
	if done, err := g.result.Poll(); done {
		if err != nil {
			return err
		}
		for n, v := range g.pixels {
			if want := []byte{byte(123 + g.reads), 0, 0, 255}[n%4]; v != want {
				return fmt.Errorf("pixel %d: got %d, want %d", n, v, want)
			}
		}
		g.reads++
		if g.reads < 6 {
			g.result = nil
			return nil
		}
		img := ebiten.NewImage(1, 1)
		img.Fill(color.White)
		g.result = img.ReadPixelsAsync(make([]byte, 4))
		return ebiten.Termination
	}
	return nil
}

func (*game) Draw(*ebiten.Image) {
}

func (*game) Layout(int, int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowVisible(false)
	g := &game{}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	const waiters = 8
	done := make(chan error, waiters)
	for range waiters {
		go func() { done <- g.result.Wait() }()
	}
	for range waiters {
		select {
		case err := <-done:
			if err == nil {
				panic("unsubmitted read-back succeeded")
			}
			if again := g.result.Wait(); again != err {
				panic("read-back error changed")
			}
			if complete, again := g.result.Poll(); !complete || again != err {
				panic("poll disagrees with wait")
			}
		case <-time.After(5 * time.Second):
			panic("unsubmitted read-back was not aborted on exit")
		}
	}
}

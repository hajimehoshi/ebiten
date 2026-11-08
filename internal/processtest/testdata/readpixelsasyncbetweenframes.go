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

func readBackOutsideGame(count int) error {
	img := ebiten.NewImage(1, 1)
	pixels := make([]byte, 4)
	for i := range count {
		want := color.RGBA{R: byte(i), G: 0x80, A: 0xff}
		img.Fill(want)
		result := img.ReadPixelsAsync(pixels)
		img.Fill(color.White)
		if err := result.Wait(); err != nil {
			return err
		}
		if got := (color.RGBA{R: pixels[0], G: pixels[1], B: pixels[2], A: pixels[3]}); got != want {
			return fmt.Errorf("read-back %d: got %v, want %v", i, got, want)
		}
	}
	return nil
}

type game struct {
	errCh chan error
}

func (g *game) Update() error {
	if g.errCh == nil {
		g.errCh = make(chan error, 1)
		go func() {
			g.errCh <- readBackOutsideGame(60)
		}()
		return nil
	}
	select {
	case err := <-g.errCh:
		if err != nil {
			return err
		}
		return ebiten.Termination
	default:
		return nil
	}
}

func (*game) Draw(*ebiten.Image) {
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

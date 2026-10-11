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
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	frames int
	err    error
}

func (g *game) Update() error {
	if g.err != nil {
		return g.err
	}
	if ebiten.IsWindowVisible() {
		return fmt.Errorf("headless window is visible")
	}
	switch g.frames {
	case 1:
		ebiten.SetWindowVisible(true)
		if ebiten.IsWindowVisible() {
			return fmt.Errorf("pending visibility exposed headless window")
		}
		ebiten.SetFullscreen(true)
		ebiten.MaximizeWindow()
		ebiten.MinimizeWindow()
	case 2:
		if ebiten.IsFullscreen() {
			return fmt.Errorf("headless window is fullscreen")
		}
		ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum)
	case 120:
		return ebiten.Termination
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	c := color.RGBA{R: uint8(g.frames), G: 123, A: 255}
	screen.Fill(c)
	var pixels [4 * 16 * 16]byte
	screen.ReadPixels(pixels[:])
	if pixels[0] != c.R || pixels[1] != c.G || pixels[3] != c.A {
		g.err = fmt.Errorf("headless readback: got %v, want %v", pixels[:4], c)
	}
	g.frames++
}

func (*game) Layout(int, int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowSize(64, 64)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetRunnableOnUnfocused(false)
	ebiten.SetFullscreen(true)
	ebiten.MaximizeWindow()
	start := time.Now()
	if err := ebiten.RunGameWithOptions(&game{}, &ebiten.RunGameOptions{Headless: true, SingleThread: os.Getenv("HEADLESS_SINGLE_THREAD") != ""}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("120 headless frames: %v\n", time.Since(start))
}

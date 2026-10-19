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
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	step int
	mode int
}

func (g *Game) Update() error {
	modes := []ebiten.WindowResizingModeType{
		ebiten.WindowResizingModeDisabled,
		ebiten.WindowResizingModeOnlyFullscreenEnabled,
	}
	switch g.step {
	case 0:
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		ebiten.MaximizeWindow()
	case 1:
		if !ebiten.IsWindowMaximized() {
			return fmt.Errorf("window was not maximized")
		}
		ebiten.SetWindowResizingMode(modes[g.mode])
	case 2:
		if ebiten.IsWindowMaximized() {
			return fmt.Errorf("IsWindowMaximized returned true with resizing disabled")
		}
		ebiten.RestoreWindow()
	case 3:
		if w, h := ebiten.WindowSize(); w != 320 || h != 240 {
			return fmt.Errorf("restored size: got %dx%d, want 320x240", w, h)
		}
		g.mode++
		if g.mode == len(modes) {
			return ebiten.Termination
		}
		g.step = -1
	}
	g.step++
	return nil
}

func (*Game) Draw(*ebiten.Image) {
}

func (*Game) Layout(int, int) (int, int) {
	return 64, 64
}

func main() {
	if runtime.GOOS != "windows" {
		return
	}
	ebiten.SetWindowVisible(false)
	ebiten.SetWindowSize(320, 240)
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}

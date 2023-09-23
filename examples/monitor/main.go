// Copyright 2023 The Ebitengine Authors
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

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	windowWidth  = 640
	windowHeight = 480
)

type Game struct {
	monitors []*ebiten.MonitorType
}

func (g *Game) Update() error {
	// Refresh monitors.
	g.monitors = ebiten.AppendMonitors(g.monitors[:0])

	// Handle keypresses.
	if inpututil.IsKeyJustReleased(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	} else {
		for i, m := range g.monitors {
			if inpututil.IsKeyJustPressed(ebiten.KeyDigit0 + ebiten.Key(i)) {
				ebiten.SetWindowTitle(m.Name())
				ebiten.SetMonitor(m)
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	lines := []string{"F to toggle fullscreen", "0-9 to change monitor"}

	lines = append(lines, "")
	for i, m := range g.monitors {
		lines = append(lines, fmt.Sprintf("%d: %s", i, m.Name()))
	}

	activeMonitor := ebiten.Monitor()
	lines = append(lines, "")
	for i, m := range g.monitors {
		if m == activeMonitor {
			lines = append(lines, fmt.Sprintf("active: %s (%d)", m.Name(), i))
		}
	}

	ebitenutil.DebugPrint(screen, strings.Join(lines, "\n"))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowWidth / 2, windowHeight / 2
}

func main() {
	g := &Game{}

	// Allow the user to pass in a monitor flag to target a specific monitor.
	var monitor int
	flag.IntVar(&monitor, "monitor", 0, "target monitor index to run the program on")
	flag.Parse()

	// Read our monitors.
	g.monitors = ebiten.AppendMonitors(nil)

	// Ensure the user did not supply a monitor index beyond the range of available monitors. If they did, set the monitor to the primary.
	if monitor < 0 || monitor >= len(g.monitors) {
		monitor = 0
	}

	targetMonitor := g.monitors[monitor]
	ebiten.SetMonitor(targetMonitor)
	ebiten.SetWindowTitle(targetMonitor.Name())
	ebiten.SetWindowSize(windowWidth, windowHeight)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

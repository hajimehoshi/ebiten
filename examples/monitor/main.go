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
	"image"
	"log"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	windowWidth  = 640
	windowHeight = 480
)

type Game struct {
	debugui debugui.DebugUI

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

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Monitors", image.Rect(10, 10, 410, 410), func(layout debugui.ContainerLayout) {
			fullscreen := ebiten.IsFullscreen()
			ctx.Checkbox(&fullscreen, "Fullscreen").On(func() {
				ebiten.SetFullscreen(fullscreen)
			})

			activeMonitor := ebiten.Monitor()
			index := -1
			for i, m := range g.monitors {
				if m == activeMonitor {
					index = i
					break
				}
			}

			ctx.Header("Active Monitor Info", true, func() {
				ctx.SetGridLayout([]int{-1, -2}, nil)
				ctx.Text("Index")
				ctx.Text(fmt.Sprintf("%d", index))
				ctx.Text("Name")
				ctx.Text(activeMonitor.Name())
				ctx.Text("Size")
				w, h := activeMonitor.Size()
				ctx.Text(fmt.Sprintf("%d x %d", w, h))
				ctx.Text("Device Scale Factor")
				ctx.Text(fmt.Sprintf("%0.2f", activeMonitor.DeviceScaleFactor()))
			})

			ctx.Header("Monitors", true, func() {
				for i, m := range g.monitors {
					ctx.IDScope(fmt.Sprintf("%d", i), func() {
						name := fmt.Sprintf("%d: %s", i, m.Name())
						if i == index {
							name += " (Active)"
						}
						ctx.TreeNode(name, func() {
							if index != i {
								ctx.Button("Activate").On(func() {
									ebiten.SetMonitor(m)
								})
							}
							ctx.SetGridLayout([]int{-1, -2}, nil)
							ctx.Text("Name")
							ctx.Text(m.Name())
							ctx.Text("Size")
							w, h := m.Size()
							ctx.Text(fmt.Sprintf("%d x %d", w, h))
							ctx.Text("Device Scale Factor")
							ctx.Text(fmt.Sprintf("%0.2f", m.DeviceScaleFactor()))
						})
					})
				}
			})
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
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

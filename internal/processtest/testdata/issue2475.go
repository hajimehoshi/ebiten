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

//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	count int
	x     int
	y     int
}

func delta(x0, x1 int) int {
	if x0 < x1 {
		return x1 - x0
	}
	return x0 - x1
}

func (g *Game) Update() error {
	switch g.count {
	case 0:
		g.x, g.y = ebiten.CursorPosition()
	case 20:
		ebiten.SetFullscreen(true)
	case 40:
		ebiten.SetFullscreen(false)
	case 60:
		return ebiten.Termination
	default:
		// Allow some numerical errors (±1).
		if x, y := ebiten.CursorPosition(); delta(g.x, x) > 1 || delta(g.y, y) > 1 {
			return fmt.Errorf("cursor position changed: got: (%d, %d), want: (%d, %d)", x, y, g.x, g.y)
		}
	}
	g.count++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	x, y := ebiten.CursorPosition()
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%d, %d", x, y))
}

func (g *Game) Layout(width, height int) (int, int) {
	// Using a fixed size matters.
	// If a window size is changed or fullscreened, the cursor position calculation considers the current screen scale, and
	// a fixed size changes the scale.
	return 320, 240
}

// runningUnderParallelsLinux reports whether the process is running inside a
// Parallels virtual machine on Linux.
func runningUnderParallelsLinux() bool {
	// The detection below relies on Linux-only interfaces.
	if runtime.GOOS != "linux" {
		return false
	}

	// systemd-detect-virt is checked first as it is reliable even on ARM guests,
	// where the DMI tables can be sparse.
	if out, err := exec.Command("systemd-detect-virt").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "parallels" {
			return true
		}
	}

	for _, name := range []string{
		"/sys/class/dmi/id/sys_vendor",
		"/sys/class/dmi/id/product_name",
	} {
		b, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		for _, f := range strings.Fields(string(b)) {
			if strings.EqualFold(f, "parallels") {
				return true
			}
		}
	}
	return false
}

func main() {
	// Mouse is not supported on mobiles.
	// Capturing a cursor requires a user gesture on browsers.
	// Skip the test in these environments.
	if runtime.GOOS == "android" || runtime.GOOS == "ios" || runtime.GOOS == "js" {
		return
	}

	// Capturing the cursor relies on the OS honoring pointer warping while
	// nothing else moves the pointer. Some environments break this contract and
	// make the captured cursor position jump, so skip the test there:
	//
	//   - Windows is flaky, especially on GitHub Actions.
	//   - Parallels drives the guest pointer as an absolute device via host mouse
	//     integration, which fights the pointer grab-and-warp that capturing uses.
	if runtime.GOOS == "windows" || runningUnderParallelsLinux() {
		return
	}

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}

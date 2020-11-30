// Copyright 2015 Hajime Hoshi
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

// +build example

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	flagFullscreen        = flag.Bool("fullscreen", false, "fullscreen")
	flagResizable         = flag.Bool("resizable", false, "make the window resizable")
	flagWindowPosition    = flag.String("windowposition", "", "window position (e.g., 100,200)")
	flagScreenTransparent = flag.Bool("screentransparent", false, "screen transparent")
	flagAutoAdjusting     = flag.Bool("autoadjusting", false, "make the game screen auto-adjusting")
	flagFloating          = flag.Bool("floating", false, "make the window floating")
	flagMaximize          = flag.Bool("maximize", false, "maximize the window")
	flagVsync             = flag.Bool("vsync", true, "enable vsync")
	flagInitFocused       = flag.Bool("initfocused", true, "whether the window is focused on start")
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
}

const (
	initScreenWidth  = 480
	initScreenHeight = 360
	initScreenScale  = 1
)

var (
	gophersImage *ebiten.Image
	count        = 0
)

func createRandomIconImage() image.Image {
	const size = 32

	r := byte(rand.Intn(0x100))
	g := byte(rand.Intn(0x100))
	b := byte(rand.Intn(0x100))
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			img.Pix[j*img.Stride+4*i] = r
			img.Pix[j*img.Stride+4*i+1] = g
			img.Pix[j*img.Stride+4*i+2] = b
			img.Pix[j*img.Stride+4*i+3] = byte(float64(i+j) / float64(2*size) * 0xff)
		}
	}

	return img
}

type game struct {
	width       int
	height      int
	transparent bool
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if *flagAutoAdjusting {
		g.width, g.height = outsideWidth, outsideHeight
		return outsideWidth, outsideHeight
	}
	// Ignore the outside size. This means that the offscreen is not adjusted with the outside world.
	return g.width, g.height
}

func (g *game) Update() error {
	var (
		screenWidth  int
		screenHeight int
		screenScale  float64
	)
	screenWidth = g.width
	screenHeight = g.height
	if ww, wh := ebiten.WindowSize(); ww > 0 && wh > 0 {
		screenScale = math.Min(float64(ww)/float64(g.width), float64(wh)/float64(g.height))
	} else {
		// ebiten.WindowSize can return (0, 0) on browsers or mobiles.
		screenScale = 1
	}

	fullscreen := ebiten.IsFullscreen()
	runnableOnUnfocused := ebiten.IsRunnableOnUnfocused()
	cursorVisible := ebiten.CursorMode() == ebiten.CursorModeVisible
	vsyncEnabled := ebiten.IsVsyncEnabled()
	tps := ebiten.MaxTPS()
	decorated := ebiten.IsWindowDecorated()
	positionX, positionY := ebiten.WindowPosition()
	g.transparent = ebiten.IsScreenTransparent()
	floating := ebiten.IsWindowFloating()
	resizable := ebiten.IsWindowResizable()
	screenCleared := ebiten.IsScreenClearedEveryFrame()

	const d = 16
	toUpdateWindowSize := false
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			screenHeight += d
			toUpdateWindowSize = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			if 16 < screenHeight && d < screenHeight {
				screenHeight -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			if 16 < screenWidth && d < screenWidth {
				screenWidth -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			screenWidth += d
			toUpdateWindowSize = true
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			positionY -= d
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			positionY += d
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			positionX -= d
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			positionX += d
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !*flagAutoAdjusting {
		switch {
		case screenScale < 1:
			screenScale = 1
		case screenScale < 1.5:
			screenScale = 1.5
		case screenScale < 2:
			screenScale = 2
		default:
			screenScale = 0.75
		}
		toUpdateWindowSize = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		fullscreen = !fullscreen
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		runnableOnUnfocused = !runnableOnUnfocused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		cursorVisible = !cursorVisible
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		vsyncEnabled = !vsyncEnabled
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		switch tps {
		case ebiten.UncappedTPS:
			tps = 30
		case 30:
			tps = 60
		case 60:
			tps = 120
		case 120:
			tps = ebiten.UncappedTPS
		default:
			panic("not reached")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		decorated = !decorated
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		floating = !floating
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		resizable = !resizable
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		screenCleared = !screenCleared
	}
	maximize := inpututil.IsKeyJustPressed(ebiten.KeyM)
	minimize := inpututil.IsKeyJustPressed(ebiten.KeyN)
	restore := false
	if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
		restore = inpututil.IsKeyJustPressed(ebiten.KeyE)
	}

	if toUpdateWindowSize {
		g.width = screenWidth
		g.height = screenHeight
		ebiten.SetWindowSize(int(float64(screenWidth)*screenScale), int(float64(screenHeight)*screenScale))
	}
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetRunnableOnUnfocused(runnableOnUnfocused)
	if cursorVisible {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}

	// Set vsync enabled only when this is needed.
	// This makes a bug around vsync initialization more explicit (#1364).
	if vsyncEnabled != ebiten.IsVsyncEnabled() {
		ebiten.SetVsyncEnabled(vsyncEnabled)
	}
	ebiten.SetMaxTPS(tps)
	ebiten.SetWindowDecorated(decorated)
	ebiten.SetWindowPosition(positionX, positionY)
	ebiten.SetWindowFloating(floating)
	ebiten.SetScreenClearedEveryFrame(screenCleared)
	if maximize && ebiten.IsWindowResizable() {
		ebiten.MaximizeWindow()
	}
	if minimize {
		ebiten.MinimizeWindow()
	}
	if restore {
		ebiten.RestoreWindow()
	}
	ebiten.SetWindowResizable(resizable)

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})
	}

	count++
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	w, h := gophersImage.Size()
	w2, h2 := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-w+w2)/2, float64(-h+h2)/2)
	dx := math.Cos(2*math.Pi*float64(count)/360) * 20
	dy := math.Sin(2*math.Pi*float64(count)/360) * 20
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(gophersImage, op)

	wx, wy := ebiten.WindowPosition()
	cx, cy := ebiten.CursorPosition()
	tpsStr := "Uncapped"
	if t := ebiten.MaxTPS(); t != ebiten.UncappedTPS {
		tpsStr = fmt.Sprintf("%d", t)
	}

	var lines []string
	if !ebiten.IsWindowMaximized() && ebiten.IsWindowResizable() {
		lines = append(lines, "[M] Maximize the window (only for desktops)")
	}
	if !ebiten.IsWindowMinimized() {
		lines = append(lines, "[N] Minimize the window (only for desktops)")
	}
	if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
		lines = append(lines, "[E] Restore the window from maximized/minimized state (only for desktops)")
	}
	msgM := strings.Join(lines, "\n")

	msgR := "[R] Switch the window resizable state (only for desktops)\n"
	fg := "Yes"
	if !ebiten.IsFocused() {
		fg = "No"
	}

	msg := fmt.Sprintf(`[Arrow keys] Move the window
[Shift + Arrow keys] Change the window size
%s
[F] Switch the fullscreen state (only for desktops)
[U] Switch the runnable-on-unfocused state
[C] Switch the cursor visibility
[I] Change the window icon (only for desktops)
[V] Switch vsync
[T] Switch TPS (ticks per second)
[D] Switch the window decoration (only for desktops)
[L] Switch the window floating state (only for desktops)
[W] Switch whether to skip clearing the screen
%s
IsFocused?: %s
Windows Position: (%d, %d)
Cursor: (%d, %d)
TPS: Current: %0.2f / Max: %s
FPS: %0.2f
Device Scale Factor: %0.2f`, msgM, msgR, fg, wx, wy, cx, cy, ebiten.CurrentTPS(), tpsStr, ebiten.CurrentFPS(), ebiten.DeviceScaleFactor())
	ebitenutil.DebugPrint(screen, msg)
}

func parseWindowPosition() (int, int, bool) {
	if *flagWindowPosition == "" {
		return 0, 0, false
	}
	tokens := strings.Split(*flagWindowPosition, ",")
	if len(tokens) != 2 {
		return 0, 0, false
	}
	x, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, 0, false
	}
	y, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, 0, false
	}
	return x, y, true
}

func main() {
	fmt.Printf("Device scale factor: %0.2f\n", ebiten.DeviceScaleFactor())
	w, h := ebiten.ScreenSizeInFullscreen()
	fmt.Printf("Screen size in fullscreen: %d, %d\n", w, h)

	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})

	if x, y, ok := parseWindowPosition(); ok {
		ebiten.SetWindowPosition(x, y)
	}
	ebiten.SetScreenTransparent(*flagScreenTransparent)

	g := &game{
		width:  initScreenWidth,
		height: initScreenHeight,
	}

	if *flagFullscreen {
		ebiten.SetFullscreen(true)
	}
	if *flagResizable {
		ebiten.SetWindowResizable(true)
	}
	if *flagFloating {
		ebiten.SetWindowFloating(true)
	}
	if *flagMaximize {
		ebiten.SetWindowResizable(true)
		ebiten.MaximizeWindow()
	}
	ebiten.SetVsyncEnabled(*flagVsync)
	if *flagAutoAdjusting {
		ebiten.SetWindowResizable(true)
	}

	ebiten.SetInitFocused(*flagInitFocused)
	if !*flagInitFocused {
		ebiten.SetRunnableOnUnfocused(true)
	}

	const title = "Window Size (Ebiten Demo)"
	ww := int(float64(g.width) * initScreenScale)
	wh := int(float64(g.height) * initScreenScale)
	ebiten.SetWindowSize(ww, wh)
	ebiten.SetWindowTitle(title)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

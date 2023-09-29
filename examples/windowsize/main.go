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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	flagFullscreen          = flag.Bool("fullscreen", false, "fullscreen")
	flagResizable           = flag.Bool("resizable", false, "make the window resizable")
	flagWindowPosition      = flag.String("windowposition", "", "window position (e.g., 100,200)")
	flagTransparent         = flag.Bool("transparent", false, "screen transparent")
	flagAutoAdjusting       = flag.Bool("autoadjusting", false, "make the game screen auto-adjusting")
	flagFloating            = flag.Bool("floating", false, "make the window floating")
	flagMaximize            = flag.Bool("maximize", false, "maximize the window")
	flagVsync               = flag.Bool("vsync", true, "enable vsync")
	flagAutoRestore         = flag.Bool("autorestore", false, "restore the window automatically")
	flagInitFocused         = flag.Bool("initfocused", true, "whether the window is focused on start")
	flagMinWindowSize       = flag.String("minwindowsize", "", "minimum window size (e.g., 100x200)")
	flagMaxWindowSize       = flag.String("maxwindowsize", "", "maximium window size (e.g., 1920x1080)")
	flagGraphicsLibrary     = flag.String("graphicslibrary", "", "graphics library (e.g. opengl)")
	flagRunnableOnUnfocused = flag.Bool("runnableonunfocused", true, "whether the app is runnable even on unfocused")
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
}

const (
	initScreenWidth  = 480
	initScreenHeight = 480
	initScreenScale  = 1
)

var (
	gophersImage *ebiten.Image
)

func createRandomIconImage() image.Image {
	const size = 32

	rf := float64(rand.Intn(0x100))
	gf := float64(rand.Intn(0x100))
	bf := float64(rand.Intn(0x100))
	img := ebiten.NewImage(size, size)
	pix := make([]byte, 4*size*size)
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			af := float64(i+j) / float64(2*size)
			if af > 0 {
				pix[4*(j*size+i)] = byte(rf * af)
				pix[4*(j*size+i)+1] = byte(gf * af)
				pix[4*(j*size+i)+2] = byte(bf * af)
				pix[4*(j*size+i)+3] = byte(af * 0xff)
			}
		}
	}
	img.WritePixels(pix)

	return img
}

type game struct {
	count       int
	width       float64
	height      float64
	transparent bool

	logOnce sync.Once
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// As game implements the interface LayoutFer, Layout is never called and LayoutF is called instead.
	// However, game has to implement Layout to satisfy the interface Game.
	panic("windowsize: Layout must not be called")
}

func (g *game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if *flagAutoAdjusting {
		g.width, g.height = outsideWidth, outsideHeight
		return outsideWidth, outsideHeight
	}
	// Ignore the outside size. This means that the offscreen is not adjusted with the outside world.
	return g.width, g.height
}

func (g *game) Update() error {
	g.logOnce.Do(func() {
		var debug ebiten.DebugInfo
		ebiten.ReadDebugInfo(&debug)
		fmt.Printf("Graphics library: %s\n", debug.GraphicsLibrary)
	})

	var (
		screenWidth  float64
		screenHeight float64
		screenScale  float64
	)
	screenWidth = g.width
	screenHeight = g.height
	if ww, wh := ebiten.WindowSize(); ww > 0 && wh > 0 {
		screenScale = math.Min(float64(ww)/g.width, float64(wh)/g.height)
	} else {
		// ebiten.WindowSize can return (0, 0) on browsers or mobiles.
		screenScale = 1
	}

	fullscreen := ebiten.IsFullscreen()
	runnableOnUnfocused := ebiten.IsRunnableOnUnfocused()
	cursorMode := ebiten.CursorMode()
	vsyncEnabled := ebiten.IsVsyncEnabled()
	tps := ebiten.TPS()
	decorated := ebiten.IsWindowDecorated()
	positionX, positionY := ebiten.WindowPosition()
	g.transparent = ebiten.IsScreenTransparent()
	floating := ebiten.IsWindowFloating()
	resizingMode := ebiten.WindowResizingMode()
	screenCleared := ebiten.IsScreenClearedEveryFrame()
	mousePassthrough := ebiten.IsWindowMousePassthrough()

	const d = 16
	toUpdateWindowSize := false
	toUpdateWindowPosition := false
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			screenHeight += d
			toUpdateWindowSize = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			if 16 < screenHeight && d < screenHeight {
				screenHeight -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			if 16 < screenWidth && d < screenWidth {
				screenWidth -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			screenWidth += d
			toUpdateWindowSize = true
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			positionY -= d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			positionY += d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			positionX -= d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			positionX += d
			toUpdateWindowPosition = true
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
		switch cursorMode {
		case ebiten.CursorModeVisible:
			cursorMode = ebiten.CursorModeHidden
		case ebiten.CursorModeHidden:
			cursorMode = ebiten.CursorModeCaptured
		case ebiten.CursorModeCaptured:
			cursorMode = ebiten.CursorModeVisible
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		vsyncEnabled = !vsyncEnabled
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		switch tps {
		case ebiten.SyncWithFPS:
			tps = 30
		case 30:
			tps = 60
		case 60:
			tps = 120
		case 120:
			tps = ebiten.SyncWithFPS
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
		switch resizingMode {
		case ebiten.WindowResizingModeDisabled:
			resizingMode = ebiten.WindowResizingModeOnlyFullscreenEnabled
		case ebiten.WindowResizingModeOnlyFullscreenEnabled:
			resizingMode = ebiten.WindowResizingModeEnabled
		case ebiten.WindowResizingModeEnabled:
			resizingMode = ebiten.WindowResizingModeDisabled
		default:
			panic("not reached")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		screenCleared = !screenCleared
	}
	maximize := inpututil.IsKeyJustPressed(ebiten.KeyM)
	minimize := inpututil.IsKeyJustPressed(ebiten.KeyN)
	restore := false
	if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
		if *flagAutoRestore {
			restore = g.count%ebiten.TPS() == 0
		} else {
			restore = inpututil.IsKeyJustPressed(ebiten.KeyE)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		mousePassthrough = !mousePassthrough
	}

	if toUpdateWindowSize {
		g.width = screenWidth
		g.height = screenHeight
		ebiten.SetWindowSize(int(float64(screenWidth)*screenScale), int(float64(screenHeight)*screenScale))
	}
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetRunnableOnUnfocused(runnableOnUnfocused)
	ebiten.SetCursorMode(cursorMode)

	// Set FPS mode enabled only when this is needed.
	// This makes a bug around FPS mode initialization more explicit (#1364).
	if vsyncEnabled != ebiten.IsVsyncEnabled() {
		ebiten.SetVsyncEnabled(vsyncEnabled)
	}
	ebiten.SetTPS(tps)
	ebiten.SetWindowDecorated(decorated)
	if toUpdateWindowPosition {
		ebiten.SetWindowPosition(positionX, positionY)
	}
	ebiten.SetWindowFloating(floating)
	ebiten.SetScreenClearedEveryFrame(screenCleared)
	if maximize && ebiten.WindowResizingMode() == ebiten.WindowResizingModeEnabled {
		ebiten.MaximizeWindow()
	}
	if minimize {
		ebiten.MinimizeWindow()
	}
	if restore {
		ebiten.RestoreWindow()
	}
	ebiten.SetWindowResizingMode(resizingMode)

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		ebiten.SetWindowIcon(nil)
	}

	ebiten.SetWindowMousePassthrough(mousePassthrough)

	g.count++
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	w2, h2 := screen.Bounds().Dx(), screen.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-w+w2)/2, float64(-h+h2)/2)
	dx := math.Cos(2*math.Pi*float64(g.count)/360) * 20
	dy := math.Sin(2*math.Pi*float64(g.count)/360) * 20
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(gophersImage, op)

	wx, wy := ebiten.WindowPosition()
	ww, wh := ebiten.WindowSize()
	minw, minh, maxw, maxh := ebiten.WindowSizeLimits()
	cx, cy := ebiten.CursorPosition()
	tpsStr := "Sync with FPS"
	if t := ebiten.TPS(); t != ebiten.SyncWithFPS {
		tpsStr = fmt.Sprintf("%d", t)
	}

	var lines []string
	if !ebiten.IsWindowMaximized() && ebiten.WindowResizingMode() == ebiten.WindowResizingModeEnabled {
		lines = append(lines, "[M] Maximize the window (only for desktops)")
	}
	if !ebiten.IsWindowMinimized() {
		lines = append(lines, "[N] Minimize the window (only for desktops)")
	}
	if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
		lines = append(lines, "[E] Restore the window from maximized/minimized state (only for desktops)")
	}
	msgM := strings.Join(lines, "\n")

	msgR := "[R] Switch the window resizing mode (only for desktops)\n"
	fg := "Yes"
	if !ebiten.IsFocused() {
		fg = "No"
	}

	msg := fmt.Sprintf(`[Arrow keys] Move the window
[Shift + Arrow keys] Change the window size
%s
[F] Switch the fullscreen state
[U] Switch the runnable-on-unfocused state
[C] Switch the cursor mode (visible, hidden, or captured)
[I] Change the window icon (only for desktops)
[J] Reset the window icon (only for desktops)
[V] Switch the vsync
[T] Switch TPS (ticks per second)
[D] Switch the window decoration (only for desktops)
[L] Switch the window floating state (only for desktops)
[W] Switch whether to skip clearing the screen
[P] Switch whether a mouse cursor passthroughs the window (only for desktops)
%s
IsFocused?: %s
Window Position: (%d, %d)
Window Size: (%d, %d)
Window size limitation: (%d, %d) - (%d, %d)
Cursor: (%d, %d)
TPS: Current: %0.2f / Max: %s
FPS: %0.2f
Device Scale Factor: %0.2f`, msgM, msgR, fg, wx, wy, ww, wh, minw, minh, maxw, maxh, cx, cy, ebiten.ActualTPS(), tpsStr, ebiten.ActualFPS(), ebiten.DeviceScaleFactor())
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

	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})

	if x, y, ok := parseWindowPosition(); ok {
		ebiten.SetWindowPosition(x, y)
	}

	g := &game{
		width:  initScreenWidth,
		height: initScreenHeight,
	}

	if *flagFullscreen {
		ebiten.SetFullscreen(true)
	}
	if *flagResizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	if *flagFloating {
		ebiten.SetWindowFloating(true)
	}
	if *flagMaximize {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		ebiten.MaximizeWindow()
	}
	if !*flagVsync {
		ebiten.SetVsyncEnabled(false)
	}
	if *flagAutoAdjusting {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	if !*flagRunnableOnUnfocused {
		ebiten.SetRunnableOnUnfocused(false)
	}

	minw, minh, maxw, maxh := -1, -1, -1, -1
	reSize := regexp.MustCompile(`^(\d+)x(\d+)$`)
	if m := reSize.FindStringSubmatch(*flagMinWindowSize); m != nil {
		minw, _ = strconv.Atoi(m[1])
		minh, _ = strconv.Atoi(m[2])
	}
	if m := reSize.FindStringSubmatch(*flagMaxWindowSize); m != nil {
		maxw, _ = strconv.Atoi(m[1])
		maxh, _ = strconv.Atoi(m[2])
	}
	if minw >= 0 || minh >= 0 || maxw >= 0 || maxh >= 0 {
		ebiten.SetWindowSizeLimits(minw, minh, maxw, maxh)
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}

	op := &ebiten.RunGameOptions{}
	switch *flagGraphicsLibrary {
	case "":
		op.GraphicsLibrary = ebiten.GraphicsLibraryAuto
	case "opengl":
		op.GraphicsLibrary = ebiten.GraphicsLibraryOpenGL
	case "directx":
		op.GraphicsLibrary = ebiten.GraphicsLibraryDirectX
	case "metal":
		op.GraphicsLibrary = ebiten.GraphicsLibraryMetal
	default:
		log.Fatalf("unexpected graphics library: %s", *flagGraphicsLibrary)
	}
	op.InitUnfocused = !*flagInitFocused
	op.ScreenTransparent = *flagTransparent

	const title = "Window Size (Ebitengine Demo)"
	ww := int(float64(g.width) * initScreenScale)
	wh := int(float64(g.height) * initScreenScale)
	ebiten.SetWindowSize(ww, wh)
	ebiten.SetWindowTitle(title)
	if err := ebiten.RunGameWithOptions(g, op); err != nil {
		log.Fatal(err)
	}
}

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
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	flagFullscreen          = flag.Bool("fullscreen", false, "fullscreen")
	flagResizable           = flag.Bool("resizable", false, "make the window resizable")
	flagWindowPosition      = flag.String("windowposition", "", "window position (e.g., 100,200)")
	flagTransparent         = flag.Bool("transparent", false, "screen transparent")
	flagFloating            = flag.Bool("floating", false, "make the window floating")
	flagMaximize            = flag.Bool("maximize", false, "maximize the window")
	flagVsync               = flag.Bool("vsync", true, "enable vsync")
	flagInitFocused         = flag.Bool("initfocused", true, "whether the window is focused on start")
	flagMinWindowSize       = flag.String("minwindowsize", "", "minimum window size (e.g., 100x200)")
	flagMaxWindowSize       = flag.String("maxwindowsize", "", "maximum window size (e.g., 1920x1080)")
	flagGraphicsLibrary     = flag.String("graphicslibrary", "", "graphics library (e.g. opengl)")
	flagRunnableOnUnfocused = flag.Bool("runnableonunfocused", true, "whether the app is runnable even on unfocused")
	flagColorSpace          = flag.String("colorspace", "", "color space ('', 'srgb', or 'display-p3')")
)

func init() {
	flag.Parse()
}

const (
	initScreenWidth  = 640
	initScreenHeight = 640
	initScreenScale  = 1
)

var (
	gophersImage *ebiten.Image
)

func createRandomIconImage() image.Image {
	const size = 32

	rf := float64(rand.IntN(0x100))
	gf := float64(rand.IntN(0x100))
	bf := float64(rand.IntN(0x100))
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
	debugUI debugui.DebugUI

	count        int
	screenWidth  float64
	screenHeight float64
	positionX    int
	positionY    int

	autoRestore         bool
	autoAdjustment      bool
	fullscreen          bool
	runnableOnUnfocused bool
	cursorMode          ebiten.CursorModeType
	vsyncEnabled        bool
	tps                 int
	decorated           bool
	floating            bool
	resizingMode        ebiten.WindowResizingModeType
	screenCleared       bool
	mousePassthrough    bool
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// As game implements the interface LayoutFer, Layout is never called and LayoutF is called instead.
	// However, game has to implement Layout to satisfy the interface Game.
	panic("windowsize: Layout must not be called")
}

func (g *game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if g.autoAdjustment {
		g.screenWidth, g.screenHeight = outsideWidth, outsideHeight
		return outsideWidth, outsideHeight
	}
	// Ignore the outside size. This means that the offscreen is not adjusted with the outside world.
	return g.screenWidth, g.screenHeight
}

func windowResigingModeString(m ebiten.WindowResizingModeType) string {
	switch m {
	case ebiten.WindowResizingModeDisabled:
		return "Disabled"
	case ebiten.WindowResizingModeOnlyFullscreenEnabled:
		return "Fullscreen Only"
	case ebiten.WindowResizingModeEnabled:
		return "Enabled"
	default:
		panic("not reached")
	}
}

func cursorModeString(m ebiten.CursorModeType) string {
	switch m {
	case ebiten.CursorModeVisible:
		return "Visible"
	case ebiten.CursorModeHidden:
		return "Hidden"
	case ebiten.CursorModeCaptured:
		return "Captured"
	default:
		panic("not reached")
	}
}

func (g *game) Update() error {
	g.fullscreen = ebiten.IsFullscreen()
	g.runnableOnUnfocused = ebiten.IsRunnableOnUnfocused()
	g.cursorMode = ebiten.CursorMode()
	g.vsyncEnabled = ebiten.IsVsyncEnabled()
	g.tps = ebiten.TPS()
	g.decorated = ebiten.IsWindowDecorated()
	g.positionX, g.positionY = ebiten.WindowPosition()
	g.floating = ebiten.IsWindowFloating()
	g.resizingMode = ebiten.WindowResizingMode()
	g.screenCleared = ebiten.IsScreenClearedEveryFrame()
	g.mousePassthrough = ebiten.IsWindowMousePassthrough()

	// ebiten.WindowSize can return (0, 0) on browsers or mobiles.
	screenScale := 1.0
	if ww, wh := ebiten.WindowSize(); ww > 0 && wh > 0 {
		screenScale = math.Min(float64(ww)/g.screenWidth, float64(wh)/g.screenHeight)
	}

	// Call SetWindowSize and SetWindowPosition only when necessary.
	// When a window is maximized, SetWindowSize and SetWindowPosition should not be called.
	// Otherwise, the restored window size and position are not correct.
	var (
		toUpdateWindowSize     bool
		toUpdateWindowPosition bool
	)

	if err := g.debugUI.Update(func(ctx *debugui.Context) error {
		ctx.Window("Window Size", image.Rect(10, 10, 330, 490), func(layout debugui.ContainerLayout) {
			ctx.Header("Instructions", false, func() {
				ctx.SetGridLayout([]int{-1, -1}, nil)
				ctx.Text("[Arrow keys]")
				ctx.Text("Move the window")
				ctx.Text("[Shift + Arrow keys]")
				ctx.Text("Change the window size")
				ctx.Text("[E]")
				ctx.Text("Restore the window (only when the window is maximized or minimized)")
			})
			ctx.Header("Settings (Window, Desktop Only)", true, func() {
				ctx.SetGridLayout([]int{-2, -1}, nil)

				ctx.Text("Resizing Mode")
				if ctx.Button(windowResigingModeString(g.resizingMode)) {
					switch g.resizingMode {
					case ebiten.WindowResizingModeDisabled:
						g.resizingMode = ebiten.WindowResizingModeOnlyFullscreenEnabled
					case ebiten.WindowResizingModeOnlyFullscreenEnabled:
						g.resizingMode = ebiten.WindowResizingModeEnabled
					case ebiten.WindowResizingModeEnabled:
						g.resizingMode = ebiten.WindowResizingModeDisabled
					default:
						panic("not reached")
					}
				}

				if ebiten.WindowResizingMode() == ebiten.WindowResizingModeEnabled {
					ctx.Text("Maxmize")
					if ctx.Button("Maximize") {
						ebiten.MaximizeWindow()
					}
				}
				ctx.Text("Minimize")
				if ctx.Button("Minimize") {
					ebiten.MinimizeWindow()
				}
				ctx.Text("Auto Restore")
				ctx.Checkbox(&g.autoRestore, "")
				if !g.autoAdjustment {
					ctx.Text("Screen Scale")
					if ctx.Button(fmt.Sprintf("%0.2f", screenScale)) {
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
				}

				ctx.Text("Decorated")
				ctx.Checkbox(&g.decorated, "")
				ctx.Text("Floating")
				ctx.Checkbox(&g.floating, "")

				ctx.Text("Update Icon")
				if ctx.Button("Update") {
					ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})
				}
				ctx.Text("Reset Icon")
				if ctx.Button("Reset") {
					ebiten.SetWindowIcon(nil)
				}
			})
			ctx.Header("Settings (Rendering)", true, func() {
				ctx.SetGridLayout([]int{-2, -1}, nil)

				ctx.Text("Auto Size Adjustment")
				ctx.Checkbox(&g.autoAdjustment, "")
				ctx.Text("Fullscreen")
				ctx.Checkbox(&g.fullscreen, "")
				ctx.Text("Runnable on Unfocused")
				ctx.Checkbox(&g.runnableOnUnfocused, "")
				ctx.Text("Vsync")
				ctx.Checkbox(&g.vsyncEnabled, "")
				ctx.Text("TPS Mode")
				tpsStr := "Sync w/ FPS"
				if t := ebiten.TPS(); t != ebiten.SyncWithFPS {
					tpsStr = fmt.Sprintf("%d", t)
				}
				if ctx.Button(tpsStr) {
					switch g.tps {
					case ebiten.SyncWithFPS:
						g.tps = 30
					case 30:
						g.tps = 60
					case 60:
						g.tps = 120
					case 120:
						g.tps = ebiten.SyncWithFPS
					default:
						panic("not reached")
					}
				}
				ctx.Text("Clear Screen Every Frame")
				ctx.Checkbox(&g.screenCleared, "")
			})
			ctx.Header("Settings (Mouse Cursor)", true, func() {
				ctx.SetGridLayout([]int{-2, -1}, nil)

				ctx.Text("Mode [C]")
				if ctx.Button(cursorModeString(g.cursorMode)) || inpututil.IsKeyJustPressed(ebiten.KeyC) {
					switch g.cursorMode {
					case ebiten.CursorModeVisible:
						g.cursorMode = ebiten.CursorModeHidden
					case ebiten.CursorModeHidden:
						g.cursorMode = ebiten.CursorModeCaptured
					case ebiten.CursorModeCaptured:
						g.cursorMode = ebiten.CursorModeVisible
					}
				}

				ctx.Text("Passthrough (desktop only) [P]")
				ctx.Checkbox(&g.mousePassthrough, "")
				if inpututil.IsKeyJustPressed(ebiten.KeyP) {
					g.mousePassthrough = !g.mousePassthrough
				}
			})
			ctx.Header("Info", true, func() {
				ctx.SetGridLayout([]int{-2, -1}, nil)

				ctx.Text("Window Position:")
				wx, wy := ebiten.WindowPosition()
				ctx.Text(fmt.Sprintf("(%d, %d)", wx, wy))

				ctx.Text("Window Size:")
				ww, wh := ebiten.WindowSize()
				ctx.Text(fmt.Sprintf("(%d, %d)", ww, wh))

				minw, minh, maxw, maxh := ebiten.WindowSizeLimits()
				ctx.Text("Minimum Window Size:")
				ctx.Text(fmt.Sprintf("(%d, %d)", minw, minh))
				ctx.Text("Maximum Window Size:")
				ctx.Text(fmt.Sprintf("(%d, %d)", maxw, maxh))

				ctx.Text("Cursor:")
				cx, cy := ebiten.CursorPosition()
				ctx.Text(fmt.Sprintf("(%d, %d)", cx, cy))

				ctx.Text("Device Scale Factor:")
				ctx.Text(fmt.Sprintf("%0.2f", ebiten.Monitor().DeviceScaleFactor()))

				w, h := ebiten.Monitor().Size()
				ctx.Text("Screen Size in Fullscreen:")
				ctx.Text(fmt.Sprintf("(%d, %d)", w, h))

				ctx.Text("Focused?")
				ctx.Text(fmt.Sprintf("%t", ebiten.IsFocused()))

				ctx.Text("TPS:")
				ctx.Text(fmt.Sprintf("%0.2f", ebiten.ActualTPS()))

				ctx.Text("FPS:")
				ctx.Text(fmt.Sprintf("%0.2f", ebiten.ActualFPS()))
			})
			ctx.Header("Info (Debug)", true, func() {
				ctx.SetGridLayout([]int{-1, -1}, nil)

				var debug ebiten.DebugInfo
				ebiten.ReadDebugInfo(&debug)

				ctx.Text("Graphics Lib:")
				ctx.Text(debug.GraphicsLibrary.String())

				ctx.Text("GPU Memory Usage:")
				ctx.Text(fmt.Sprintf("%d", debug.TotalGPUImageMemoryUsageInBytes))
			})
		})
		return nil
	}); err != nil {
		return err
	}

	const d = 16
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.screenHeight += d
			toUpdateWindowSize = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			if 16 < g.screenHeight && d < g.screenHeight {
				g.screenHeight -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			if 16 < g.screenWidth && d < g.screenWidth {
				g.screenWidth -= d
				toUpdateWindowSize = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.screenWidth += d
			toUpdateWindowSize = true
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.positionY -= d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.positionY += d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.positionX -= d
			toUpdateWindowPosition = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.positionX += d
			toUpdateWindowPosition = true
		}
	}

	var restore bool
	if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
		if g.autoRestore {
			restore = g.count%ebiten.TPS() == 0
		} else {
			restore = inpututil.IsKeyJustPressed(ebiten.KeyE)
		}
	}

	if toUpdateWindowSize {
		ebiten.SetWindowSize(int(g.screenWidth*screenScale), int(g.screenHeight*screenScale))
	}
	ebiten.SetFullscreen(g.fullscreen)
	ebiten.SetRunnableOnUnfocused(g.runnableOnUnfocused)
	ebiten.SetCursorMode(g.cursorMode)

	// Set FPS mode enabled only when this is needed.
	// This makes a bug around FPS mode initialization more explicit (#1364).
	if g.vsyncEnabled != ebiten.IsVsyncEnabled() {
		ebiten.SetVsyncEnabled(g.vsyncEnabled)
	}
	ebiten.SetTPS(g.tps)
	ebiten.SetWindowDecorated(g.decorated)
	if toUpdateWindowPosition {
		ebiten.SetWindowPosition(g.positionX, g.positionY)
	}
	ebiten.SetWindowFloating(g.floating)
	ebiten.SetScreenClearedEveryFrame(g.screenCleared)
	if restore {
		ebiten.RestoreWindow()
	}
	ebiten.SetWindowResizingMode(g.resizingMode)

	ebiten.SetWindowMousePassthrough(g.mousePassthrough)

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

	g.debugUI.Draw(screen)
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
		screenWidth:  initScreenWidth,
		screenHeight: initScreenHeight,
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
	switch *flagColorSpace {
	case "":
		op.ColorSpace = ebiten.ColorSpaceDefault
	case "srgb":
		op.ColorSpace = ebiten.ColorSpaceSRGB
	case "display-p3":
		op.ColorSpace = ebiten.ColorSpaceDisplayP3
	}
	op.InitUnfocused = !*flagInitFocused
	op.ScreenTransparent = *flagTransparent
	op.X11ClassName = "Window-Size"
	op.X11InstanceName = "window-size"

	const title = "Window Size (Ebitengine Demo)"
	ww := int(float64(g.screenWidth) * initScreenScale)
	wh := int(float64(g.screenHeight) * initScreenScale)
	ebiten.SetWindowSize(ww, wh)
	ebiten.SetWindowTitle(title)
	if err := ebiten.RunGameWithOptions(g, op); err != nil {
		log.Fatal(err)
	}
}

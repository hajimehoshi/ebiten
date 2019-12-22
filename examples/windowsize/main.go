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

// +build example jsgo

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
)

var (
	// flagLegacy represents whether the legacy APIs are used or not.
	// If flagLegacy is true, these legacy APIs are used:
	//
	//   * ebiten.Run
	//   * ebiten.ScreenScale
	//   * ebiten.SetScreenScale
	//   * ebiten.SetScreenSize
	//
	// If flagLegacy is false, these APIs are used:
	//
	//   * ebiten.RunGame
	//   * ebiten.SetWindowSize
	//   * ebiten.WindowSize
	//
	// A resizable window is available only when flagLegacy is false.
	flagLegacy = flag.Bool("legacy", false, "use the legacy API")

	flagFullscreen        = flag.Bool("fullscreen", false, "fullscreen")
	flagWindowPosition    = flag.String("windowposition", "", "window position (e.g., 100,200)")
	flagScreenTransparent = flag.Bool("screentransparent", false, "screen transparent")
	flagAutoAdjusting     = flag.Bool("autoadjusting", false, "make the game screen auto-adjusting")
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
	width  int
	height int
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if *flagAutoAdjusting {
		g.width, g.height = outsideWidth, outsideHeight
		return outsideWidth, outsideHeight
	}
	// Ignore the outside size. This means that the offscreen is not adjusted with the outside world.
	return g.width, g.height
}

func (g *game) Update(screen *ebiten.Image) error {
	var (
		screenWidth  int
		screenHeight int
		screenScale  float64
	)
	if *flagLegacy {
		screenWidth, screenHeight = screen.Size()
		screenScale = ebiten.ScreenScale()
	} else {
		screenWidth = g.width
		screenHeight = g.height
		if ww, wh := ebiten.WindowSize(); ww > 0 && wh > 0 {
			screenScale = math.Min(float64(ww)/float64(g.width), float64(wh)/float64(g.height))
		} else {
			// ebiten.WindowSize can return (0, 0) on browsers or mobiles.
			screenScale = 1
		}
	}

	fullscreen := ebiten.IsFullscreen()
	runnableInBackground := ebiten.IsRunnableInBackground()
	cursorVisible := ebiten.IsCursorVisible()
	vsyncEnabled := ebiten.IsVsyncEnabled()
	tps := ebiten.MaxTPS()
	decorated := ebiten.IsWindowDecorated()
	positionX, positionY := ebiten.WindowPosition()
	transparent := ebiten.IsScreenTransparent()
	resizable := ebiten.IsWindowResizable()

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
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		runnableInBackground = !runnableInBackground
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
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		resizable = !resizable
	}

	if toUpdateWindowSize {
		if *flagLegacy {
			ebiten.SetScreenSize(screenWidth, screenHeight)
			ebiten.SetScreenScale(screenScale)
		} else {
			g.width = screenWidth
			g.height = screenHeight
			ebiten.SetWindowSize(int(float64(screenWidth)*screenScale), int(float64(screenHeight)*screenScale))
		}
	}
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetRunnableInBackground(runnableInBackground)
	ebiten.SetCursorVisible(cursorVisible)
	ebiten.SetVsyncEnabled(vsyncEnabled)
	ebiten.SetMaxTPS(tps)
	ebiten.SetWindowDecorated(decorated)
	ebiten.SetWindowPosition(positionX, positionY)
	if !*flagLegacy {
		// A resizable window is available only with RunGame.
		ebiten.SetWindowResizable(resizable)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})
	}

	count++

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	if !transparent {
		screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	}
	w, h := gophersImage.Size()
	w2, h2 := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-w+w2)/2, float64(-h+h2)/2)
	dx := math.Cos(2*math.Pi*float64(count)/360) * 10
	dy := math.Sin(2*math.Pi*float64(count)/360) * 10
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(gophersImage, op)

	wx, wy := ebiten.WindowPosition()
	cx, cy := ebiten.CursorPosition()
	tpsStr := "Uncapped"
	if t := ebiten.MaxTPS(); t != ebiten.UncappedTPS {
		tpsStr = fmt.Sprintf("%d", t)
	}

	var msgS string
	var msgR string
	if *flagLegacy {
		msgS = "Press S key to change the window scale (only for desktops)\n"
	} else {
		msgR = "Press R key to switch the window resizable state (only for desktops)\n"
	}

	msg := fmt.Sprintf(`Press arrow keys to move the window
Press shift + arrow keys to change the window size
%sPress F key to switch the fullscreen state (only for desktops)
Press B key to switch the run-in-background state
Press C key to switch the cursor visibility
Press I key to change the window icon (only for desktops)
Press V key to switch vsync
Press T key to switch TPS (ticks per second)
Press D key to switch the window decoration (only for desktops)
%sWindows Position: (%d, %d)
Cursor: (%d, %d)
TPS: Current: %0.2f / Max: %s
FPS: %0.2f
Device Scale Factor: %0.2f`, msgS, msgR, wx, wy, cx, cy, ebiten.CurrentTPS(), tpsStr, ebiten.CurrentFPS(), ebiten.DeviceScaleFactor())
	ebitenutil.DebugPrint(screen, msg)
	return nil
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

	if !*flagLegacy {
		fmt.Println("Tip: With -autoadjusting flag, you can make an adjustable game screen.")
	}

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
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

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
	if *flagAutoAdjusting {
		if *flagLegacy {
			log.Println("-autoadjusting flag cannot work with -legacy flag")
		}
		ebiten.SetWindowResizable(true)
	}

	const title = "Window Size (Ebiten Demo)"
	if *flagLegacy {
		if err := ebiten.Run(g.Update, g.width, g.height, initScreenScale, title); err != nil {
			log.Fatal(err)
		}
	} else {
		w := int(float64(g.width) * initScreenScale)
		h := int(float64(g.height) * initScreenScale)
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowTitle(title)
		if err := ebiten.RunGame(g); err != nil {
			log.Fatal(err)
		}
	}
}

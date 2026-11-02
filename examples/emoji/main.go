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

package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

// fontURL is a URL of Noto Color Emoji, a color emoji font using COLRv1 and SVG glyph data.
// The license is the SIL Open Font License, Version 1.1:
//
// https://fonts.google.com/noto/specimen/Noto+Color+Emoji
// Copyright 2022 Google Inc.
const fontURL = "https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf"

const sampleText = "😀🍤🍣🍵😅\n🖐🏻🖐🏼🖐🏽🖐🏾🖐🏿"

type Game struct {
	debugui        debugui.DebugUI
	highResolution bool
	scale          float64

	faceSourceCh chan *text.GoTextFaceSource
	faces        []*text.GoTextFace
	fontPath     string
}

// loadingMessage returns a message shown while the font is being loaded.
func (g *Game) loadingMessage() string {
	// Cycle "", ".", "..", and "..." every 0.5 seconds.
	dots := strings.Repeat(".", int(ebiten.Tick()/int64(ebiten.TPS()/2)%4))
	if g.fontPath != "" {
		return "Loading the font" + dots
	}
	return "Downloading the font" + dots + "\nYou can specify a font file as an argument instead."
}

func (g *Game) Update() error {
	highResolutionAvailable := ebiten.Monitor().DeviceScaleFactor() > 1
	if !highResolutionAvailable {
		g.highResolution = false
	}
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.SetScale(max(1, int(math.Round(g.scale))))
		ctx.Window("Emoji", image.Rect(420, 10, 630, 80), func(layout debugui.ContainerLayout) {
			if highResolutionAvailable {
				ctx.Checkbox(&g.highResolution, "High resolution")
			} else {
				ctx.Text("High resolution unavailable")
			}
		})
		return nil
	}); err != nil {
		return err
	}

	if len(g.faces) == 0 {
		// Use select and 'default' clause for non-blocking receiving.
		select {
		case src := <-g.faceSourceCh:
			for _, size := range []float64{24, 48, 96} {
				g.faces = append(g.faces, &text.GoTextFace{
					Source: src,
					Size:   size,
				})
			}
		default:
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if len(g.faces) == 0 {
		ebitenutil.DebugPrint(screen, g.loadingMessage())
		g.debugui.Draw(screen)
		return
	}

	y := 8 * g.scale
	for _, baseFace := range g.faces {
		face := *baseFace
		face.Size *= g.scale
		op := &text.DrawOptions{}
		op.GeoM.Translate(8*g.scale, y)
		op.LineSpacing = face.Size * 1.25
		// Do not set a color scale: a color scale would also scale the colors of emojis.
		text.Draw(screen, sampleText, &face, op)
		y += op.LineSpacing * 2
	}
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("emoji: Layout must not be called")
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	g.scale = 1
	if g.highResolution {
		g.scale = ebiten.Monitor().DeviceScaleFactor()
		g.scale *= outsideWidth / screenWidth
	}
	return screenWidth * g.scale, screenHeight * g.scale
}

// loadFaceSource loads a font from the file at path.
// If path is empty, the font is downloaded from fontURL.
func loadFaceSource(path string) (*text.GoTextFaceSource, error) {
	var in io.ReadCloser
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		in = f
	} else {
		res, err := http.Get(fontURL)
		if err != nil {
			return nil, err
		}
		if res.StatusCode < 200 || 300 <= res.StatusCode {
			_ = res.Body.Close()
			return nil, fmt.Errorf("fetching %s failed: %s", fontURL, res.Status)
		}
		in = res.Body
	}
	fontData, err := io.ReadAll(in)
	_ = in.Close()
	if err != nil {
		return nil, err
	}
	return text.NewGoTextFaceSource(bytes.NewReader(fontData))
}

func main() {
	g := &Game{
		faceSourceCh: make(chan *text.GoTextFaceSource),
	}

	// You can specify your own emoji font file as an argument.
	// Color emoji fonts in the SVG, COLRv0, sbix, and CBDT formats are supported.
	// A COLRv1 font works when the font also has an SVG table, like Noto Color Emoji.
	if len(os.Args) > 1 {
		g.fontPath = os.Args[1]
	} else {
		fmt.Println("Downloading the font. You can specify a font file as an argument instead.")
	}

	// Loading the font might take time, so load it in parallel with the game.
	// The emojis are rendered after the loading is done.
	go func() {
		src, err := loadFaceSource(g.fontPath)
		if err != nil {
			log.Fatal(err)
		}
		g.faceSourceCh <- src
		close(g.faceSourceCh)
	}()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Emoji (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

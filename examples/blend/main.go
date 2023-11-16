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
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"

	"golang.org/x/image/font/inconsolata"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images/blend"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 800
	screenHeight = 800
)

var (
	inconsolataFace = text.NewStdFace(inconsolata.Bold8x16)
)

// mode is a blend mode with description.
type mode struct {
	blend ebiten.Blend
	name  string
}

// Game is a canvas for drawing blend mode tiles.
type Game struct {
	source    *ebiten.Image
	dest      *ebiten.Image
	offscreen *ebiten.Image
	tileSize  int
	modes     []mode
}

func NewGame() (*Game, error) {
	source, err := loadImage(blend.Source_png)
	if err != nil {
		return nil, fmt.Errorf("fail to load source: %w", err)
	}
	dest, err := loadImage(blend.Dest_png)
	if err != nil {
		return nil, fmt.Errorf("fail to load dest: %w", err)
	}

	// Set up a grid for drawing.
	g := &Game{
		source: source,
		dest:   dest,
	}

	// Add all known blend modes and their names.
	g.modes = []mode{
		{blend: ebiten.BlendCopy, name: "BlendCopy"},
		{blend: ebiten.BlendSourceAtop, name: "BlendSourceAtop"},
		{blend: ebiten.BlendSourceOver, name: "BlendSourceOver"},
		{blend: ebiten.BlendSourceIn, name: "BlendSourceIn"},
		{blend: ebiten.BlendSourceOut, name: "BlendSourceOut"},
		{blend: ebiten.BlendDestination, name: "BlendDestination"},
		{blend: ebiten.BlendDestinationAtop, name: "BlendDestinationAtop"},
		{blend: ebiten.BlendDestinationOver, name: "BlendDestinationOver"},
		{blend: ebiten.BlendDestinationIn, name: "BlendDestinationIn"},
		{blend: ebiten.BlendDestinationOut, name: "BlendDestinationOut"},
		{blend: ebiten.BlendXor, name: "BlendXor"},
		{blend: ebiten.BlendLighter, name: "BlendLighter"},
		{blend: ebiten.BlendClear, name: "BlendClear"},
	}

	// Adjust the tile size for the available images.
	g.tileSize = maxSide(g.source, g.dest)
	g.offscreen = ebiten.NewImage(g.tileSize, g.tileSize)

	return g, nil
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	const (
		tileGap = 64
		textGap = 16
		gridW   = 4
	)

	// Clear the screen.
	screen.Fill(color.White)

	// Get an offset for center alignment.
	// Where sw, sh is the screen size in pixels.
	// And gw, gh is the grid size in pixels.
	totalTileSize := g.tileSize + tileGap
	gridH := (len(g.modes)-1)/gridW + 1
	gw, gh := gridW*totalTileSize-tileGap, gridH*totalTileSize
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	ox, oy := float64(sw-gw)/2, float64(sh-gh)/2

	// Draw a tilemap.
	for i, m := range g.modes {
		x := i % gridW
		y := i / gridW

		px, py := x*totalTileSize, y*totalTileSize

		// Making a place for the text.
		py += textGap * y

		// Drawing the blend mode and it's name.
		alignedX, alignedY := float64(px)+ox, float64(py)+oy
		g.drawBlendMode(screen, alignedX, alignedY, m.blend)
		drawCenteredText(screen, alignedX+float64(g.tileSize)/2, alignedY+float64(g.tileSize)+textGap, m.name)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) drawBlendMode(screen *ebiten.Image, x, y float64, mode ebiten.Blend) {
	// Copy the destination image to offscreen so as not to modify it.
	g.offscreen.Clear()
	g.offscreen.DrawImage(g.dest, nil)

	// Select and apply blending mode.
	op := &ebiten.DrawImageOptions{}
	op.Blend = mode
	g.offscreen.DrawImage(g.source, op)

	// Draw the result on the passed coordinates.
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(g.offscreen, op)
}

// loadImage is a util function for loading embedded images.
func loadImage(data []byte) (*ebiten.Image, error) {
	m, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return ebiten.NewImageFromImage(m), nil
}

// max returns the largest of x or y.
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// maxSide returns the largest side of a or b images.
func maxSide(a, b *ebiten.Image) int {
	return max(
		max(a.Bounds().Dx(), b.Bounds().Dx()),
		max(a.Bounds().Dy(), b.Bounds().Dy()),
	)
}

// drawCenteredText is a util function for drawing blend mode description.
func drawCenteredText(screen *ebiten.Image, cx, cy float64, s string) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(color.Black)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(screen, s, inconsolataFace, op)
}

func main() {
	ebiten.SetWindowSize(640, 640)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Blend modes (Ebitengine Demo)")

	game, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}

	if err = ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

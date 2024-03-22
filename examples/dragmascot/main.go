// Copyright 2024 The Ebitengine Authors
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

// Mascot is a desktop mascot on cross platforms.
// This is inspired by mattn's gopher (https://github.com/mattn/gopher).
package main

import (
	"bytes"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	rmascot "github.com/hajimehoshi/ebiten/v2/examples/resources/images/mascot"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	width  = 200
	height = 200
)

var (
	gopher1 *ebiten.Image
	gopher2 *ebiten.Image
	gopher3 *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img1, _, err := image.Decode(bytes.NewReader(rmascot.Out01_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher1 = ebiten.NewImageFromImage(img1)

	img2, _, err := image.Decode(bytes.NewReader(rmascot.Out02_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher2 = ebiten.NewImageFromImage(img2)

	img3, _, err := image.Decode(bytes.NewReader(rmascot.Out03_png))
	if err != nil {
		log.Fatal(err)
	}
	gopher3 = ebiten.NewImageFromImage(img3)
}

type mascot struct {
	count int

	dragging         bool
	dragStartWindowX int
	dragStartWindowY int
	dragStartCursorX int
	dragStartCursorY int

	cursorToWindowX float64
	cursorToWindowY float64
}

func (m *mascot) Layout(outsideWidth, outsideHeight int) (int, int) {
	// The cursor position is in a "logical" coordinate, which is determined by the game width and height.
	// Calculate the factors to convert a cursor position to a window position.
	m.cursorToWindowX = float64(outsideWidth) / float64(width)
	m.cursorToWindowY = float64(outsideHeight) / float64(height)
	return width, height
}

func (m *mascot) Update() error {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		m.dragging = false
	}
	if !m.dragging && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		m.dragging = true
		m.dragStartWindowX, m.dragStartWindowY = ebiten.WindowPosition()
		m.dragStartCursorX, m.dragStartCursorY = ebiten.CursorPosition()
	}
	if m.dragging {
		// Move the window only by the delta of the cursor.
		cx, cy := ebiten.CursorPosition()
		dx := int(float64(cx-m.dragStartCursorX) * m.cursorToWindowX)
		dy := int(float64(cy-m.dragStartCursorY) * m.cursorToWindowY)
		wx, wy := ebiten.WindowPosition()
		ebiten.SetWindowPosition(wx+dx, wy+dy)
		m.count++
	}

	return nil
}

func (m *mascot) Draw(screen *ebiten.Image) {
	img := gopher1
	if m.dragging {
		switch (m.count / 3) % 4 {
		case 0:
			img = gopher1
		case 1, 3:
			img = gopher2
		case 2:
			img = gopher3
		}
	}
	screen.DrawImage(img, nil)
}

func main() {
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowSize(width, height)

	op := &ebiten.RunGameOptions{}
	op.ScreenTransparent = true
	op.SkipTaskbar = true
	if err := ebiten.RunGameWithOptions(&mascot{}, op); err != nil {
		log.Fatal(err)
	}
}

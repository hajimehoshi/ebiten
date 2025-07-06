// Copyright 2021 The Ebiten Authors
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
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	grids      map[image.Rectangle]ebiten.CursorShapeType
	gridColors map[image.Rectangle]color.Color
	cursor     ebiten.CursorShapeType
}

func (g *Game) Update() error {
	pt := image.Pt(ebiten.CursorPosition())
	cursor := ebiten.CursorShapeDefault
	for r, c := range g.grids {
		if pt.In(r) {
			cursor = c
			break
		}
	}

	// Call SetCursorShape only when this is changed to test Ebitengine remembers the current cursor correctly even when it is hidden.
	if g.cursor != cursor {
		ebiten.SetCursorShape(cursor)
		g.cursor = cursor
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		switch ebiten.CursorMode() {
		case ebiten.CursorModeVisible:
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		case ebiten.CursorModeHidden:
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for r, c := range g.gridColors {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}

	switch ebiten.CursorShape() {
	case ebiten.CursorShapeDefault:
		ebitenutil.DebugPrint(screen, "CursorShape: Default")
	case ebiten.CursorShapeText:
		ebitenutil.DebugPrint(screen, "CursorShape: Text")
	case ebiten.CursorShapeCrosshair:
		ebitenutil.DebugPrint(screen, "CursorShape: Crosshair")
	case ebiten.CursorShapePointer:
		ebitenutil.DebugPrint(screen, "CursorShape: Pointer")
	case ebiten.CursorShapeEWResize:
		ebitenutil.DebugPrint(screen, "CursorShape: EW Resize")
	case ebiten.CursorShapeNSResize:
		ebitenutil.DebugPrint(screen, "CursorShape: NS Resize")
	case ebiten.CursorShapeNESWResize:
		ebitenutil.DebugPrint(screen, "CursorShape: NESW Resize")
	case ebiten.CursorShapeNWSEResize:
		ebitenutil.DebugPrint(screen, "CursorShape: NWSE Resize")
	case ebiten.CursorShapeMove:
		ebitenutil.DebugPrint(screen, "CursorShape: Move")
	case ebiten.CursorShapeNotAllowed:
		ebitenutil.DebugPrint(screen, "CursorShape: Not Allowed")
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{
		grids: map[image.Rectangle]ebiten.CursorShapeType{
			image.Rect(100, 100, 200, 200): ebiten.CursorShapeDefault,
			image.Rect(200, 100, 300, 200): ebiten.CursorShapeText,
			image.Rect(300, 100, 400, 200): ebiten.CursorShapeCrosshair,
			image.Rect(400, 100, 500, 200): ebiten.CursorShapePointer,
			image.Rect(100, 200, 200, 300): ebiten.CursorShapeEWResize,
			image.Rect(200, 200, 300, 300): ebiten.CursorShapeNSResize,
			image.Rect(300, 200, 400, 300): ebiten.CursorShapeNESWResize,
			image.Rect(400, 200, 500, 300): ebiten.CursorShapeNWSEResize,
			image.Rect(100, 300, 200, 400): ebiten.CursorShapeMove,
			image.Rect(200, 300, 300, 400): ebiten.CursorShapeNotAllowed,
		},
		gridColors: map[image.Rectangle]color.Color{},
	}
	for rect, c := range g.grids {
		clr := color.RGBA{0x40, 0x40, 0x40, 0xff}
		switch c % 3 {
		case 0:
			clr.R = 0x80
		case 1:
			clr.G = 0x80
		case 2:
			clr.B = 0x80
		}
		g.gridColors[rect] = clr
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Cursor (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

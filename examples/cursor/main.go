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

// +build example

package main

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	grids      map[image.Rectangle]ebiten.CursorShapeType
	gridColors map[image.Rectangle]color.Color
}

func (g *Game) Update() error {
	pt := image.Pt(ebiten.CursorPosition())
	for r, c := range g.grids {
		if pt.In(r) {
			ebiten.SetCursorShape(c)
			return nil
		}
	}
	ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for r, c := range g.gridColors {
		ebitenutil.DrawRect(screen, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), c)
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
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{
		grids: map[image.Rectangle]ebiten.CursorShapeType{
			image.Rect(100, 100, 200, 300): ebiten.CursorShapeDefault,
			image.Rect(200, 100, 300, 300): ebiten.CursorShapeText,
			image.Rect(300, 100, 400, 300): ebiten.CursorShapeCrosshair,
			image.Rect(400, 100, 500, 300): ebiten.CursorShapePointer,
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
	ebiten.SetWindowTitle("Cursor (Ebiten Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

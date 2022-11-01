// Copyright 2014 Hajime Hoshi
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
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	brushImage *ebiten.Image
)

func init() {
	const (
		a0 = 0x40
		a1 = 0xc0
		a2 = 0xff
	)
	pixels := []uint8{
		a0, a1, a1, a0,
		a1, a2, a2, a1,
		a1, a2, a2, a1,
		a0, a1, a1, a0,
	}
	brushImage = ebiten.NewImageFromImage(&image.Alpha{
		Pix:    pixels,
		Stride: 4,
		Rect:   image.Rect(0, 0, 4, 4),
	})
}

type touch struct {
	id  ebiten.TouchID
	pos pos
}

type pos struct {
	x int
	y int
}

type Game struct {
	cursor  pos
	touches []touch
	count   int

	canvasImage *ebiten.Image
}

func NewGame() *Game {
	g := &Game{
		canvasImage: ebiten.NewImage(screenWidth, screenHeight),
	}
	g.canvasImage.Fill(color.White)
	return g
}

func (g *Game) Update() error {
	drawn := false

	// Paint the brush by mouse dragging
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.paint(g.canvasImage, mx, my)
		drawn = true
	}
	g.cursor = pos{
		x: mx,
		y: my,
	}

	// Paint the brush by touches
	g.touches = g.touches[:0]
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		g.paint(g.canvasImage, x, y)
		g.touches = append(g.touches, touch{
			id: id,
			pos: pos{
				x: x,
				y: y,
			},
		})
		drawn = true
	}
	if drawn {
		g.count++
	}
	return nil
}

// paint draws the brush on the given canvas image at the position (x, y).
func (g *Game) paint(canvas *ebiten.Image, x, y int) {
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	var cm colorm.ColorM
	// Scale the color and rotate the hue so that colors vary on each frame.
	cm.Scale(1.0, 0.50, 0.125, 1.0)
	tps := ebiten.TPS()
	theta := 2.0 * math.Pi * float64(g.count%tps) / float64(tps)
	cm.RotateHue(theta)
	colorm.DrawImage(canvas, brushImage, cm, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.canvasImage, nil)

	msg := fmt.Sprintf("(%d, %d)", g.cursor.x, g.cursor.y)
	for _, t := range g.touches {
		msg += fmt.Sprintf("\n(%d, %d) touch %d", t.pos.x, t.pos.y, t.id)
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Paint (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

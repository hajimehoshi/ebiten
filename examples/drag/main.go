// Copyright 2018 The Ebiten Authors
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
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

// Sprite represents an image.
type Sprite struct {
	image *ebiten.Image
	x     int
	y     int
}

// In returns true if (x, y) is in the sprite, and false otherwise.
func (s *Sprite) In(x, y int) bool {
	// Check the actual color (alpha) value at the specified position
	// so that the result of In becomes natural to users.
	return s.image.At(x-s.x, y-s.y).(color.RGBA).A > 0
}

// MoveBy moves the sprite by (x, y).
func (s *Sprite) MoveBy(x, y int) {
	w, h := s.image.Size()

	s.x += x
	s.y += y
	if s.x < 0 {
		s.x = 0
	}
	if s.x > screenWidth-w {
		s.x = screenWidth - w
	}
	if s.y < 0 {
		s.y = 0
	}
	if s.y > screenHeight-h {
		s.y = screenHeight - h
	}
}

// Draw draws the sprite.
func (s *Sprite) Draw(screen *ebiten.Image, dx, dy int, alpha float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(s.x+dx), float64(s.y+dy))
	op.ColorM.Scale(1, 1, 1, alpha)
	screen.DrawImage(s.image, op)
	screen.DrawImage(s.image, op)
}

type DragPhase int

const (
	DragPhaseNone DragPhase = iota
	DragPhaseStart
	DragPhaseDrag
	DragPhaseEnd
)

// DragState manages the current drag state.
type DragState struct {
	phase DragPhase

	// initX and initY represents the position when dragging starts.
	// initX and initY values don't make sense when phase is DragPhaseNone.
	initX int
	initY int

	// currentX and currentY represents the current position
	// initX and initY values don't make sense when phase is DragPhaseNone.
	currentX int
	currentY int
}

func (d *DragState) Update() {
	switch d.phase {
	case DragPhaseNone:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			d.phase = DragPhaseStart
			d.initX = cx
			d.initY = cy
			d.currentX = cx
			d.currentY = cy
		}
	case DragPhaseStart:
		d.phase = DragPhaseDrag
	case DragPhaseDrag:
		x, y := ebiten.CursorPosition()
		d.currentX = x
		d.currentY = y
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			d.phase = DragPhaseEnd
		}
	case DragPhaseEnd:
		d.phase = DragPhaseNone
	}
}

type Game struct {
	dragState DragState
	sprites   []*Sprite

	// draggingSpriteIndex represents the index of the sprites
	// that is being dragged. If draggingSpriteIndex is -1,
	// there is not such sprite.
	draggingSpriteIndex int
}

var theGame *Game

func init() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage, _ := ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	// Initialize the sprites.
	sprites := []*Sprite{}
	w, h := ebitenImage.Size()
	for i := 0; i < 50; i++ {
		s := &Sprite{
			image: ebitenImage,
			x:     rand.Intn(screenWidth - w),
			y:     rand.Intn(screenHeight - h),
		}
		sprites = append(sprites, s)
	}

	// Initialize the game.
	theGame = &Game{
		sprites:             sprites,
		draggingSpriteIndex: -1,
	}
}

func (g *Game) update(screen *ebiten.Image) error {
	g.dragState.Update()
	switch g.dragState.phase {
	case DragPhaseStart:
		if g.draggingSpriteIndex == -1 {
			// As the sprites are ordered from back to front,
			// search the clicked/touched sprite in reverse order.
			for i := len(g.sprites) - 1; i >= 0; i-- {
				s := g.sprites[i]
				if s.In(g.dragState.initX, g.dragState.initY) {
					g.draggingSpriteIndex = i
					break
				}
			}
		}
	case DragPhaseEnd:
		if g.draggingSpriteIndex != -1 {
			dx := g.dragState.currentX - g.dragState.initX
			dy := g.dragState.currentY - g.dragState.initY
			g.sprites[g.draggingSpriteIndex].MoveBy(dx, dy)

			// Move the dragged sprite to the front.
			s := g.sprites[g.draggingSpriteIndex]
			g.sprites = append(
				g.sprites[:g.draggingSpriteIndex],
				g.sprites[g.draggingSpriteIndex+1:]...)
			g.sprites = append(g.sprites, s)

			g.draggingSpriteIndex = -1
		}
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	for i, s := range g.sprites {
		if i == g.draggingSpriteIndex {
			s.Draw(screen, 0, 0, 0.5)
		} else {
			s.Draw(screen, 0, 0, 1)
		}
	}
	if g.draggingSpriteIndex != -1 {
		s := g.sprites[g.draggingSpriteIndex]
		dx := g.dragState.currentX - g.dragState.initX
		dy := g.dragState.currentY - g.dragState.initY
		s.Draw(screen, dx, dy, 1)
	}

	ebitenutil.DebugPrint(screen, "Drag & Drop the sprites!")

	return nil
}

func main() {
	if err := ebiten.Run(theGame.update, screenWidth, screenHeight, 2, "Drag & Drop (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}

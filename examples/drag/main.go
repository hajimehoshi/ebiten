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

package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	screenWidth  = 640
	screenHeight = 480
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
	//
	// Note that this is not a good manner to use At for logic
	// since color from At might include some errors on some machines.
	// As this is not so important logic, it's ok to use it so far.
	return s.image.At(x-s.x, y-s.y).(color.RGBA).A > 0
}

// MoveBy moves the sprite by (x, y).
func (s *Sprite) MoveBy(x, y int) {
	w, h := s.image.Bounds().Dx(), s.image.Bounds().Dy()

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
func (s *Sprite) Draw(screen *ebiten.Image, dx, dy int, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(s.x+dx), float64(s.y+dy))
	op.ColorScale.ScaleAlpha(alpha)
	screen.DrawImage(s.image, op)
	screen.DrawImage(s.image, op)
}

// StrokeSource represents a input device to provide strokes.
type StrokeSource interface {
	Position() (int, int)
	IsJustReleased() bool
}

// MouseStrokeSource is a StrokeSource implementation of mouse.
type MouseStrokeSource struct{}

func (m *MouseStrokeSource) Position() (int, int) {
	return ebiten.CursorPosition()
}

func (m *MouseStrokeSource) IsJustReleased() bool {
	return inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
}

// TouchStrokeSource is a StrokeSource implementation of touch.
type TouchStrokeSource struct {
	ID ebiten.TouchID
}

func (t *TouchStrokeSource) Position() (int, int) {
	return ebiten.TouchPosition(t.ID)
}

func (t *TouchStrokeSource) IsJustReleased() bool {
	return inpututil.IsTouchJustReleased(t.ID)
}

// Stroke manages the current drag state by mouse.
type Stroke struct {
	source StrokeSource

	// initX and initY represents the position when dragging starts.
	initX int
	initY int

	// currentX and currentY represents the current position
	currentX int
	currentY int

	released bool

	// draggingObject represents a object (sprite in this case)
	// that is being dragged.
	draggingObject any
}

func NewStroke(source StrokeSource) *Stroke {
	cx, cy := source.Position()
	return &Stroke{
		source:   source,
		initX:    cx,
		initY:    cy,
		currentX: cx,
		currentY: cy,
	}
}

func (s *Stroke) Update() {
	if s.released {
		return
	}
	if s.source.IsJustReleased() {
		s.released = true
		return
	}
	x, y := s.source.Position()
	s.currentX = x
	s.currentY = y
}

func (s *Stroke) IsReleased() bool {
	return s.released
}

func (s *Stroke) Position() (int, int) {
	return s.currentX, s.currentY
}

func (s *Stroke) PositionDiff() (int, int) {
	dx := s.currentX - s.initX
	dy := s.currentY - s.initY
	return dx, dy
}

func (s *Stroke) DraggingObject() any {
	return s.draggingObject
}

func (s *Stroke) SetDraggingObject(object any) {
	s.draggingObject = object
}

type Game struct {
	touchIDs []ebiten.TouchID
	strokes  map[*Stroke]struct{}
	sprites  []*Sprite
}

var ebitenImage *ebiten.Image

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)
}

func NewGame() *Game {
	// Initialize the sprites.
	sprites := []*Sprite{}
	w, h := ebitenImage.Bounds().Dx(), ebitenImage.Bounds().Dy()
	for i := 0; i < 50; i++ {
		s := &Sprite{
			image: ebitenImage,
			x:     rand.Intn(screenWidth - w),
			y:     rand.Intn(screenHeight - h),
		}
		sprites = append(sprites, s)
	}

	// Initialize the game.
	return &Game{
		strokes: map[*Stroke]struct{}{},
		sprites: sprites,
	}
}

func (g *Game) spriteAt(x, y int) *Sprite {
	// As the sprites are ordered from back to front,
	// search the clicked/touched sprite in reverse order.
	for i := len(g.sprites) - 1; i >= 0; i-- {
		s := g.sprites[i]
		if s.In(x, y) {
			return s
		}
	}
	return nil
}

func (g *Game) updateStroke(stroke *Stroke) {
	stroke.Update()
	if !stroke.IsReleased() {
		return
	}

	s := stroke.DraggingObject().(*Sprite)
	if s == nil {
		return
	}

	s.MoveBy(stroke.PositionDiff())

	index := -1
	for i, ss := range g.sprites {
		if ss == s {
			index = i
			break
		}
	}

	// Move the dragged sprite to the front.
	g.sprites = append(g.sprites[:index], g.sprites[index+1:]...)
	g.sprites = append(g.sprites, s)

	stroke.SetDraggingObject(nil)
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s := NewStroke(&MouseStrokeSource{})
		s.SetDraggingObject(g.spriteAt(s.Position()))
		g.strokes[s] = struct{}{}
	}
	g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
	for _, id := range g.touchIDs {
		s := NewStroke(&TouchStrokeSource{id})
		s.SetDraggingObject(g.spriteAt(s.Position()))
		g.strokes[s] = struct{}{}
	}

	for s := range g.strokes {
		g.updateStroke(s)
		if s.IsReleased() {
			delete(g.strokes, s)
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	draggingSprites := map[*Sprite]struct{}{}
	for s := range g.strokes {
		if sprite := s.DraggingObject().(*Sprite); sprite != nil {
			draggingSprites[sprite] = struct{}{}
		}
	}

	for _, s := range g.sprites {
		if _, ok := draggingSprites[s]; ok {
			continue
		}
		s.Draw(screen, 0, 0, 1)
	}
	for s := range g.strokes {
		dx, dy := s.PositionDiff()
		if sprite := s.DraggingObject().(*Sprite); sprite != nil {
			sprite.Draw(screen, dx, dy, 0.5)
		}
	}

	ebitenutil.DebugPrint(screen, "Drag & Drop the sprites!")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Drag & Drop (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

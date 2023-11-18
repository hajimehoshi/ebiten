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
	image      *ebiten.Image
	alphaImage *image.Alpha
	x          int
	y          int
	dragged    bool
}

// In returns true if (x, y) is in the sprite, and false otherwise.
func (s *Sprite) In(x, y int) bool {
	// Check the actual color (alpha) value at the specified position
	// so that the result of In becomes natural to users.
	//
	// Use alphaImage (*image.Alpha) instead of image (*ebiten.Image) here.
	// It is because (*ebiten.Image).At is very slow as this reads pixels from GPU,
	// and should be avoided whenever possible.
	return s.alphaImage.At(x-s.x, y-s.y).(color.Alpha).A > 0
}

// MoveTo moves the sprite to the position (x, y).
func (s *Sprite) MoveTo(x, y int) {
	w, h := s.image.Bounds().Dx(), s.image.Bounds().Dy()

	s.x = x
	s.y = y
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
func (s *Sprite) Draw(screen *ebiten.Image, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(s.x), float64(s.y))
	op.ColorScale.ScaleAlpha(alpha)
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

	// offsetX and offsetY represents a relative value from the sprite's upper-left position to the cursor position.
	offsetX int
	offsetY int

	// sprite represents a sprite being dragged.
	sprite *Sprite
}

func NewStroke(source StrokeSource, sprite *Sprite) *Stroke {
	sprite.dragged = true
	x, y := source.Position()
	return &Stroke{
		source:  source,
		offsetX: x - sprite.x,
		offsetY: y - sprite.y,
		sprite:  sprite,
	}
}

func (s *Stroke) Update() {
	if !s.sprite.dragged {
		return
	}
	if s.source.IsJustReleased() {
		s.sprite.dragged = false
		return
	}

	x, y := s.source.Position()
	x -= s.offsetX
	y -= s.offsetY
	s.sprite.MoveTo(x, y)
}

func (s *Stroke) Sprite() *Sprite {
	return s.sprite
}

type Game struct {
	touchIDs []ebiten.TouchID
	strokes  map[*Stroke]struct{}
	sprites  []*Sprite
}

var (
	ebitenImage      *ebiten.Image
	ebitenAlphaImage *image.Alpha
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)

	// Clone an image but only with alpha values.
	// This is used to detect a user cursor touches the image.
	b := img.Bounds()
	ebitenAlphaImage = image.NewAlpha(b)
	for j := b.Min.Y; j < b.Max.Y; j++ {
		for i := b.Min.X; i < b.Max.X; i++ {
			ebitenAlphaImage.Set(i, j, img.At(i, j))
		}
	}
}

func NewGame() *Game {
	// Initialize the sprites.
	sprites := []*Sprite{}
	w, h := ebitenImage.Bounds().Dx(), ebitenImage.Bounds().Dy()
	for i := 0; i < 50; i++ {
		s := &Sprite{
			image:      ebitenImage,
			alphaImage: ebitenAlphaImage,
			x:          rand.Intn(screenWidth - w),
			y:          rand.Intn(screenHeight - h),
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

func (g *Game) moveSpriteToFront(sprite *Sprite) {
	index := -1
	for i, ss := range g.sprites {
		if ss == sprite {
			index = i
			break
		}
	}
	g.sprites = append(g.sprites[:index], g.sprites[index+1:]...)
	g.sprites = append(g.sprites, sprite)
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if sp := g.spriteAt(ebiten.CursorPosition()); sp != nil {
			s := NewStroke(&MouseStrokeSource{}, sp)
			g.strokes[s] = struct{}{}
			g.moveSpriteToFront(sp)
		}
	}
	g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
	for _, id := range g.touchIDs {
		if sp := g.spriteAt(ebiten.TouchPosition(id)); sp != nil {
			s := NewStroke(&TouchStrokeSource{id}, sp)
			g.strokes[s] = struct{}{}
			g.moveSpriteToFront(sp)
		}
	}

	for s := range g.strokes {
		s.Update()
		if !s.sprite.dragged {
			delete(g.strokes, s)
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, s := range g.sprites {
		if s.dragged {
			s.Draw(screen, 0.5)
		} else {
			s.Draw(screen, 1)
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

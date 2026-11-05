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

package pinch

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	buttonColor   = color.RGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xe0}
	buttonOnColor = color.RGBA{R: 0x20, G: 0x60, B: 0xc0, A: 0xe0}
)

// button is a button on the screen, which works with a mouse and a touch screen alike.
type button struct {
	label string
	rect  image.Rectangle

	// on reports whether the button is lit. on is nil for a button that is never lit.
	on func() bool

	// press is called when the button is pressed.
	press func()
}

// pressedPoints appends the points pressed at the current tick, by a mouse or fingers, to points.
func pressedPoints(points []image.Point, touchIDs []ebiten.TouchID) ([]image.Point, []ebiten.TouchID) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		points = append(points, image.Pt(ebiten.CursorPosition()))
	}
	touchIDs = inpututil.AppendJustPressedTouchIDs(touchIDs[:0])
	for _, id := range touchIDs {
		points = append(points, image.Pt(ebiten.TouchPosition(id)))
	}
	return points, touchIDs
}

func (b *button) update(points []image.Point) {
	for _, p := range points {
		if p.In(b.rect) {
			b.press()
			return
		}
	}
}

func (b *button) draw(screen *ebiten.Image) {
	clr := buttonColor
	if b.on != nil && b.on() {
		clr = buttonOnColor
	}
	r := b.rect
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), clr, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, color.White, false)

	// A character of the debug font is 6x16 pixels.
	x := r.Min.X + (r.Dx()-len(b.label)*6)/2
	y := r.Min.Y + (r.Dy()-16)/2
	ebitenutil.DebugPrintAt(screen, b.label, x, y)
}

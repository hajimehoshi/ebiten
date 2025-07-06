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

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	contentArea image.Rectangle

	// dragging indicates whether a pointing device is dragging the content.
	dragging bool

	// prevY is the previous Y value of a pointing device in the last frame.
	prevY int

	// startY is the initial Y value of a pointing device where dragging started.
	startY int

	// offsetY is the content offset in the Y axis.
	offsetY int

	// offsetStartY is the content offset in the Y axis when dragging started.
	offsetStartY int

	// velocityY is the scrolling velocity in the Y axis.
	velocityY int

	itemCount  int
	itemHeight int

	touchIDs             []ebiten.TouchID
	justReleasedTouchIDs []ebiten.TouchID
}

func (g *Game) updateInput() {
	g.touchIDs = ebiten.AppendTouchIDs(g.touchIDs[:0])
	g.justReleasedTouchIDs = inpututil.AppendJustReleasedTouchIDs(g.justReleasedTouchIDs[:0])
}

// pointingDevicePosition returns the position of a pointing device (a touch or a mouse).
func (g *Game) pointingDevicePosition() (x, y int) {
	if len(g.touchIDs) > 0 {
		return ebiten.TouchPosition(g.touchIDs[0])
	}
	return ebiten.CursorPosition()
}

// isPointingDevicePressed reports whether a pointing device is pressed.
func (g *Game) isPointingDevicePressed() bool {
	if len(g.touchIDs) > 0 {
		return true
	}
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

// isPointingDeviceJustReleased reports whether a pointing device is just released.
func (g *Game) isPointingDeviceJustReleased() bool {
	if len(g.justReleasedTouchIDs) > 0 {
		return false
	}
	return inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
}

func (g *Game) Update() error {
	g.updateInput()

	x, y := g.pointingDevicePosition()
	defer func() {
		g.prevY = y
	}()

	hovering := image.Pt(x, y).In(g.contentArea)

	// If a pointing device is just released, start scrolling.
	if g.isPointingDeviceJustReleased() && g.dragging {
		g.dragging = false
		g.velocityY = y - g.prevY
		return nil
	}

	// Process a mouse wheel.
	if _, wheelY := ebiten.Wheel(); wheelY != 0 && !g.dragging && hovering {
		g.velocityY = int(wheelY)
	}

	// If a pointing device is NOT pressed, scroll by the inertia.
	if !g.isPointingDevicePressed() {
		g.dragging = false

		g.setOffsetY(g.offsetY + g.velocityY)
		if g.velocityY != 0 {
			g.velocityY = int(float64(g.velocityY) * 15.0 / 16.0)
		}
		return nil
	}

	// As a pointing device is pressed, stop the inertia.
	g.velocityY = 0

	if !g.dragging && hovering {
		g.dragging = true
		g.offsetStartY = g.offsetY
		g.startY = y
	}

	// If a pinting device is pressed, adjust the offset by the movement.
	if g.dragging {
		g.setOffsetY(g.offsetStartY + y - g.startY)
	}

	return nil
}

func (g *Game) setOffsetY(offsetY int) {
	g.offsetY = offsetY
	h := g.contentHeight()
	if g.offsetY < -h+g.contentArea.Dy() {
		g.offsetY = -h + g.contentArea.Dy()
	}
	if g.offsetY > 0 {
		g.offsetY = 0
	}
}

func (g *Game) contentHeight() int {
	return g.itemCount * g.itemHeight
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Render the content. Use SubImage as a mask.
	screenContentArea := screen.SubImage(g.contentArea).(*ebiten.Image)

	for i := 0; i < g.itemCount; i++ {
		itemRegion := image.Rect(0, i*g.itemHeight, g.contentArea.Dx(), (i+1)*g.itemHeight)
		itemRegion = itemRegion.Add(image.Pt(g.contentArea.Min.X, g.contentArea.Min.Y))
		itemRegion = itemRegion.Add(image.Pt(0, g.offsetY))

		// Skip rendering if the item is out of the content area.
		if itemRegion.Intersect(g.contentArea).Empty() {
			continue
		}

		vector.FillRect(screenContentArea, float32(itemRegion.Min.X), float32(itemRegion.Min.Y), float32(itemRegion.Dx()), float32(itemRegion.Dy()), color.RGBA{byte(i), byte(i), byte(i), 0xff}, false)
		text := fmt.Sprintf("Item %d", i)
		if i == 0 {
			text += " (drag or touch to scroll)"
		}
		ebitenutil.DebugPrintAt(screenContentArea, text, itemRegion.Min.X, itemRegion.Min.Y)
	}

	// Render the content border line.
	vector.StrokeRect(screen, float32(g.contentArea.Min.X), float32(g.contentArea.Min.Y), float32(g.contentArea.Dx()), float32(g.contentArea.Dy()), 1, color.White, false)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	const offset = 16
	g.contentArea = image.Rect(offset, offset, outsideWidth-offset, outsideHeight-offset)
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowTitle("Scroll (Ebitengine Demo)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(320, 240, -1, -1)
	if err := ebiten.RunGame(&Game{
		itemCount:  256,
		itemHeight: 24,
	}); err != nil {
		log.Fatal(err)
	}
}

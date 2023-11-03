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
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
	w, h         int
	currentLevel *Level

	camX, camY float64
	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	offscreen *ebiten.Image
}

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
	l, err := NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	g := &Game{
		currentLevel: l,
		camScale:     1,
		camScaleTo:   1,
		mousePanX:    math.MinInt32,
		mousePanY:    math.MinInt32,
	}
	return g, nil
}

// Update reads current user input and updates the Game state.
func (g *Game) Update() error {
	// Update target zoom level.
	var scrollY float64
	if ebiten.IsKeyPressed(ebiten.KeyC) || ebiten.IsKeyPressed(ebiten.KeyPageDown) {
		scrollY = -0.25
	} else if ebiten.IsKeyPressed(ebiten.KeyE) || ebiten.IsKeyPressed(ebiten.KeyPageUp) {
		scrollY = .25
	} else {
		_, scrollY = ebiten.Wheel()
		if scrollY < -1 {
			scrollY = -1
		} else if scrollY > 1 {
			scrollY = 1
		}
	}
	g.camScaleTo += scrollY * (g.camScaleTo / 7)

	// Clamp target zoom level.
	if g.camScaleTo < 0.01 {
		g.camScaleTo = 0.01
	} else if g.camScaleTo > 100 {
		g.camScaleTo = 100
	}

	// Smooth zoom transition.
	div := 10.0
	if g.camScaleTo > g.camScale {
		g.camScale += (g.camScaleTo - g.camScale) / div
	} else if g.camScaleTo < g.camScale {
		g.camScale -= (g.camScale - g.camScaleTo) / div
	}

	// Pan camera via keyboard.
	pan := 7.0 / g.camScale
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camX -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camX += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camY -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camY += pan
	}

	// Pan camera via mouse.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if g.mousePanX == math.MinInt32 && g.mousePanY == math.MinInt32 {
			g.mousePanX, g.mousePanY = ebiten.CursorPosition()
		} else {
			x, y := ebiten.CursorPosition()
			dx, dy := float64(g.mousePanX-x)*(pan/100), float64(g.mousePanY-y)*(pan/100)
			g.camX, g.camY = g.camX-dx, g.camY+dy
		}
	} else if g.mousePanX != math.MinInt32 || g.mousePanY != math.MinInt32 {
		g.mousePanX, g.mousePanY = math.MinInt32, math.MinInt32
	}

	// Clamp camera position.
	worldWidth := float64(g.currentLevel.w * g.currentLevel.tileSize / 2)
	worldHeight := float64(g.currentLevel.h * g.currentLevel.tileSize / 2)
	if g.camX < -worldWidth {
		g.camX = -worldWidth
	} else if g.camX > worldWidth {
		g.camX = worldWidth
	}
	if g.camY < -worldHeight {
		g.camY = -worldHeight
	} else if g.camY > 0 {
		g.camY = 0
	}

	// Randomize level.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := NewLevel()
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}

		g.currentLevel = l
	}

	return nil
}

// Draw draws the Game on the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Render level.
	g.renderLevel(screen)

	// Print game info.
	ebitenutil.DebugPrint(screen, fmt.Sprintf("KEYS WASD EC R\nFPS  %0.0f\nTPS  %0.0f\nSCA  %0.2f\nPOS  %0.0f,%0.0f", ebiten.ActualFPS(), ebiten.ActualTPS(), g.camScale, g.camX, g.camY))
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.w, g.h = outsideWidth, outsideHeight
	return g.w, g.h
}

// cartesianToIso transforms cartesian coordinates into isometric coordinates.
func (g *Game) cartesianToIso(x, y float64) (float64, float64) {
	tileSize := g.currentLevel.tileSize
	ix := (x - y) * float64(tileSize/2)
	iy := (x + y) * float64(tileSize/4)
	return ix, iy
}

/*
This function might be useful for those who want to modify this example.

// isoToCartesian transforms isometric coordinates into cartesian coordinates.
func (g *Game) isoToCartesian(x, y float64) (float64, float64) {
	tileSize := g.currentLevel.tileSize
	cx := (x/float64(tileSize/2) + y/float64(tileSize/4)) / 2
	cy := (y/float64(tileSize/4) - (x / float64(tileSize/2))) / 2
	return cx, cy
}
*/

// renderLevel draws the current Level on the screen.
func (g *Game) renderLevel(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	padding := float64(g.currentLevel.tileSize) * g.camScale
	cx, cy := float64(g.w/2), float64(g.h/2)

	scaleLater := g.camScale > 1
	target := screen
	scale := g.camScale

	// When zooming in, tiles can have slight bleeding edges.
	// To avoid them, render the result on an offscreen first and then scale it later.
	if scaleLater {
		if g.offscreen != nil {
			if g.offscreen.Bounds().Size() != screen.Bounds().Size() {
				g.offscreen.Deallocate()
				g.offscreen = nil
			}
		}
		if g.offscreen == nil {
			s := screen.Bounds().Size()
			g.offscreen = ebiten.NewImage(s.X, s.Y)
		}
		target = g.offscreen
		target.Clear()
		scale = 1
	}

	for y := 0; y < g.currentLevel.h; y++ {
		for x := 0; x < g.currentLevel.w; x++ {
			xi, yi := g.cartesianToIso(float64(x), float64(y))

			// Skip drawing tiles that are out of the screen.
			drawX, drawY := ((xi-g.camX)*g.camScale)+cx, ((yi+g.camY)*g.camScale)+cy
			if drawX+padding < 0 || drawY+padding < 0 || drawX > float64(g.w) || drawY > float64(g.h) {
				continue
			}

			t := g.currentLevel.tiles[y][x]
			if t == nil {
				continue // No tile at this position.
			}

			op.GeoM.Reset()
			// Move to current isometric position.
			op.GeoM.Translate(xi, yi)
			// Translate camera position.
			op.GeoM.Translate(-g.camX, g.camY)
			// Zoom.
			op.GeoM.Scale(scale, scale)
			// Center.
			op.GeoM.Translate(cx, cy)

			t.Draw(target, op)
		}
	}

	if scaleLater {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Scale(float64(g.camScale), float64(g.camScale))
		op.GeoM.Translate(cx, cy)
		screen.DrawImage(target, op)
	}
}

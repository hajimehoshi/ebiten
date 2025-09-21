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

package ebiten

import (
	"image"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/inputstate"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var screenFilterEnabled atomic.Bool

func init() {
	screenFilterEnabled.Store(true)
}

type gameForUI struct {
	game        Game
	offscreen   *Image
	screen      *Image
	imageDumper imageDumper
	transparent bool
}

func newGameForUI(game Game, transparent bool) *gameForUI {
	g := &gameForUI{
		game:        game,
		transparent: transparent,
	}
	return g
}

func (g *gameForUI) NewOffscreenImage(width, height int) *ui.Image {
	if g.offscreen != nil {
		g.offscreen.Deallocate()
		g.offscreen = nil
	}

	// Keep the offscreen an unmanaged image that is always isolated from an atlas (#1938).
	// The shader program for the screen is special and doesn't work well with an image on an atlas.
	// An image on an atlas is surrounded by a transparent edge,
	// and the shader program unexpectedly picks the pixel on the edges.
	imageType := atlas.ImageTypeUnmanaged
	if ui.Get().IsScreenClearedEveryFrame() {
		// A volatile image is also always isolated.
		imageType = atlas.ImageTypeVolatile
	}
	g.offscreen = newImage(image.Rect(0, 0, width, height), imageType)
	return g.offscreen.image
}

func (g *gameForUI) NewScreenImage(width, height int) *ui.Image {
	if g.screen != nil {
		g.screen.Deallocate()
		g.screen = nil
	}

	g.screen = newImage(image.Rect(0, 0, width, height), atlas.ImageTypeScreen)
	return g.screen.image
}

func (g *gameForUI) Layout(outsideWidth, outsideHeight float64) (float64, float64) {
	if l, ok := g.game.(LayoutFer); ok {
		return l.LayoutF(outsideWidth, outsideHeight)
	}

	// Even if the original value is less than 1, the value must be a positive integer (#2340).
	// This is for a simple implementation of Layout, which returns the argument values without modifications.
	// TODO: Remove this hack when Game.Layout takes floats instead of integers.
	if outsideWidth < 1 {
		outsideWidth = 1
	}
	if outsideHeight < 1 {
		outsideHeight = 1
	}

	sw, sh := g.game.Layout(int(outsideWidth), int(outsideHeight))
	return float64(sw), float64(sh)
}

func (g *gameForUI) UpdateInputState(fn func(*ui.InputState)) {
	inputstate.Get().Update(fn)
}

func (g *gameForUI) Update() error {
	if err := g.game.Update(); err != nil {
		return err
	}
	if err := g.imageDumper.update(); err != nil {
		return err
	}
	return nil
}

func (g *gameForUI) DrawOffscreen() error {
	g.game.Draw(g.offscreen)
	if err := g.imageDumper.dump(g.offscreen, g.transparent); err != nil {
		return err
	}
	return nil
}

func (g *gameForUI) DrawFinalScreen(scale, offsetX, offsetY float64) {
	var geoM GeoM
	geoM.Scale(scale, scale)
	geoM.Translate(offsetX, offsetY)

	if d, ok := g.game.(FinalScreenDrawer); ok {
		d.DrawFinalScreen(g.screen, g.offscreen, geoM)
		return
	}

	DefaultDrawFinalScreen(g.screen, g.offscreen, geoM)
}

// DefaultDrawFinalScreen is the default implementation of [FinalScreenDrawer.DrawFinalScreen],
// used when a [Game] doesn't implement [FinalScreenDrawer].
//
// You can use DefaultDrawFinalScreen when you need the default implementation of [FinalScreenDrawer.DrawFinalScreen]
// in your implementation of [FinalScreenDrawer], for example.
func DefaultDrawFinalScreen(screen FinalScreen, offscreen *Image, geoM GeoM) {
	scale := geoM.Element(0, 0)
	switch {
	case !screenFilterEnabled.Load(), math.Floor(scale) == scale:
		op := &DrawImageOptions{}
		op.GeoM = geoM
		screen.DrawImage(offscreen, op)
	case scale < 1:
		op := &DrawImageOptions{}
		op.GeoM = geoM
		op.Filter = FilterLinear
		screen.DrawImage(offscreen, op)
	default:
		op := &DrawImageOptions{}
		op.GeoM = geoM
		op.Filter = FilterPixelated
		screen.DrawImage(offscreen, op)
	}
}

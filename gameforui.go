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
	"fmt"
	"image"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

const screenShaderSrc = `//kage:unit pixels

package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Blend source colors in a square region, which size is 1/scale.
	scale := imageDstSize()/imageSrc0Size()
	pos := texCoord
	p0 := pos - 1/2.0/scale
	p1 := pos + 1/2.0/scale

	// Texels must be in the source rect, so it is not necessary to check.
	c0 := imageSrc0UnsafeAt(p0)
	c1 := imageSrc0UnsafeAt(vec2(p1.x, p0.y))
	c2 := imageSrc0UnsafeAt(vec2(p0.x, p1.y))
	c3 := imageSrc0UnsafeAt(p1)

	// p is the p1 value in one pixel assuming that the pixel's upper-left is (0, 0) and the lower-right is (1, 1).
	rate := clamp(fract(p1)*scale, 0, 1)
	return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
}
`

var screenFilterEnabled = int32(1)

func isScreenFilterEnabled() bool {
	return atomic.LoadInt32(&screenFilterEnabled) != 0
}

func setScreenFilterEnabled(enabled bool) {
	v := int32(0)
	if enabled {
		v = 1
	}
	atomic.StoreInt32(&screenFilterEnabled, v)
}

type gameForUI struct {
	game         Game
	offscreen    *Image
	screen       *Image
	screenShader *Shader
	imageDumper  imageDumper
	transparent  bool
}

func newGameForUI(game Game, transparent bool) *gameForUI {
	g := &gameForUI{
		game:        game,
		transparent: transparent,
	}

	s, err := NewShader([]byte(screenShaderSrc))
	if err != nil {
		panic(fmt.Sprintf("ebiten: compiling the screen shader failed: %v", err))
	}
	g.screenShader = s

	return g
}

func (g *gameForUI) NewOffscreenImage(width, height int) *ui.Image {
	if g.offscreen != nil {
		g.offscreen.Dispose()
		g.offscreen = nil
	}

	// Keep the offscreen an unmanaged image that is always isolated from an atlas (#1938).
	// The shader program for the screen is special and doesn't work well with an image on an atlas.
	// An image on an atlas is surrounded by a transparent edge,
	// and the shader program unexpectedly picks the pixel on the edges.
	imageType := atlas.ImageTypeUnmanaged
	if ui.IsScreenClearedEveryFrame() {
		// A violatile image is also always isolated.
		imageType = atlas.ImageTypeVolatile
	}
	g.offscreen = newImage(image.Rect(0, 0, width, height), imageType)
	return g.offscreen.image
}

func (g *gameForUI) NewScreenImage(width, height int) *ui.Image {
	if g.screen != nil {
		g.screen.Dispose()
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

	// TODO: Add a new Layout function taking float values (#2285).
	sw, sh := g.game.Layout(int(outsideWidth), int(outsideHeight))
	return float64(sw), float64(sh)
}

func (g *gameForUI) UpdateInputState(fn func(*ui.InputState)) {
	theInputState.update(fn)
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

	switch {
	case !isScreenFilterEnabled(), math.Floor(scale) == scale:
		op := &DrawImageOptions{}
		op.GeoM = geoM
		g.screen.DrawImage(g.offscreen, op)
	case scale < 1:
		op := &DrawImageOptions{}
		op.GeoM = geoM
		op.Filter = FilterLinear
		g.screen.DrawImage(g.offscreen, op)
	default:
		op := &DrawRectShaderOptions{}
		op.Images[0] = g.offscreen
		op.GeoM = geoM
		w, h := g.offscreen.Bounds().Dx(), g.offscreen.Bounds().Dy()
		g.screen.DrawRectShader(w, h, g.screenShader, op)
	}
}

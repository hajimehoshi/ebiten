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
	"image/color"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/inputstate"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var screenFilterEnabled atomic.Bool

func init() {
	screenFilterEnabled.Store(true)
}

// finalScreenTransparent reports whether the screen of the running game is transparent.
// Unlike screenTransparent, it does not change after the game starts.
var finalScreenTransparent atomic.Bool

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
	finalScreenTransparent.Store(transparent)
	return g
}

// screenSize is the size of the image given at Draw.
// A nil value means that the game has not started yet.
var screenSize atomic.Pointer[image.Point]

func (g *gameForUI) NewOffscreenImage(width, height int) *ui.Image {
	bounds := imageRectWithValidation(0, 0, width, height)
	if g.offscreen != nil {
		g.offscreen.Deallocate()
		g.offscreen = nil
	}

	// Keep the offscreen an unmanaged image that is always isolated from an atlas (#1938).
	// The shader program for the screen is special and doesn't work well with an image on an atlas.
	// An image on an atlas is surrounded by a transparent edge,
	// and the shader program unexpectedly picks the pixel on the edges.
	g.offscreen = newImage(bounds, atlas.ImageTypeUnmanaged)
	screenSize.Store(&image.Point{X: width, Y: height})
	return g.offscreen.image.image
}

func (g *gameForUI) NewScreenImage(width, height int) *ui.Image {
	bounds := imageRectWithValidation(0, 0, width, height)
	if g.screen != nil {
		g.screen.Deallocate()
		g.screen = nil
	}

	g.screen = newImage(bounds, atlas.ImageTypeScreen)
	return g.screen.image.image
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
	// A synchronous pixel read cannot return its error, so the error ends the game here, including
	// one in this Update.
	if err := readPixelsError(); err != nil {
		return err
	}
	if err := g.game.Update(); err != nil {
		return err
	}
	if err := readPixelsError(); err != nil {
		return err
	}
	if err := g.imageDumper.update(); err != nil {
		return err
	}
	return nil
}

func (g *gameForUI) ClearOffscreen() {
	if g.offscreen.isDisposed() {
		return
	}
	g.offscreen.image.Fill(0, 0, 0, 0, g.offscreen.adjustedBounds())
}

func (g *gameForUI) DrawOffscreen() error {
	g.game.Draw(g.offscreen)
	if !g.offscreen.isDisposed() {
		// Apply the pending pixel writes, so that the ui package detects the modification by Draw.
		g.offscreen.image.writeBackPixelsIfNeeded()
	}
	if err := g.imageDumper.dump(g.offscreen, g.transparent); err != nil {
		return err
	}
	return nil
}

func (g *gameForUI) ClearScreen(opaque bool) {
	if g.screen.isDisposed() {
		return
	}
	var a float32
	if opaque {
		a = 1
	}
	g.screen.image.Fill(0, 0, 0, a, g.screen.adjustedBounds())
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

func (g *gameForUI) HasCustomFinalScreenDrawer() bool {
	_, ok := g.game.(FinalScreenDrawer)
	return ok
}

// DefaultDrawFinalScreen is the default implementation of [FinalScreenDrawer.DrawFinalScreen],
// used when a [Game] doesn't implement [FinalScreenDrawer].
//
// DefaultDrawFinalScreen overwrites every pixel of screen: offscreen drawn with geoM, and black elsewhere.
// If the screen is not transparent, every resulting pixel is opaque.
//
// The exact rendering result is not guaranteed to be compatible across Ebitengine versions.
// The result might also depend on Ebitengine's state, especially on whether the game loop has started.
//
// You can use DefaultDrawFinalScreen when you need the default implementation of [FinalScreenDrawer.DrawFinalScreen]
// in your implementation of [FinalScreenDrawer], for example.
func DefaultDrawFinalScreen(screen FinalScreen, offscreen *Image, geoM GeoM) {
	filter := builtinshader.FilterNearest
	scale := geoM.Element(0, 0)
	switch {
	case !screenFilterEnabled.Load(), math.Floor(scale) == scale:
	case scale < 1:
		filter = builtinshader.FilterLinear
	default:
		filter = builtinshader.FilterPixelated
	}
	opaque := !finalScreenTransparent.Load()

	// Call the methods on the concrete type. A call via the FinalScreen interface makes the options
	// escape to the heap in each frame.
	dst := screen.(*Image)

	// Place the offscreen's corners where DrawImage places them, including its snapping of the
	// destination positions, so that the pixels covered and the source positions sampled are the
	// same. Then map the destination's corners back to source positions with the inverse of the
	// affine map defined by three of the placed corners.
	a, b, c, d, tx, ty := geoM.elements32()
	srcBounds := offscreen.Bounds()
	w, h := float32(srcBounds.Dx()), float32(srcBounds.Dy())
	var corners [4 * graphics.VertexFloatCount]float32
	graphics.QuadVerticesFromSrcAndMatrix(corners[:], 0, 0, w, h, a, b, c, d, tx, ty, 1, 1, 1, 1)
	ox, oy := float64(corners[0]), float64(corners[1])
	var m00, m01, m10, m11 float64
	if w > 0 && h > 0 {
		m00 = (float64(corners[graphics.VertexFloatCount]) - ox) / float64(w)
		m10 = (float64(corners[graphics.VertexFloatCount+1]) - oy) / float64(w)
		m01 = (float64(corners[2*graphics.VertexFloatCount]) - ox) / float64(h)
		m11 = (float64(corners[2*graphics.VertexFloatCount+1]) - oy) / float64(h)
	}
	det := m00*m11 - m01*m10

	if a*d-b*c == 0 || det == 0 {
		// The offscreen covers no pixel.
		var alpha uint8
		if opaque {
			alpha = 0xff
		}
		dst.Fill(color.RGBA{A: alpha})
		return
	}

	dstBounds := dst.Bounds()
	var vertices [4]Vertex
	for j, p := range [4]image.Point{
		dstBounds.Min,
		{X: dstBounds.Max.X, Y: dstBounds.Min.Y},
		{X: dstBounds.Min.X, Y: dstBounds.Max.Y},
		dstBounds.Max,
	} {
		dx := float64(p.X) - ox
		dy := float64(p.Y) - oy
		vertices[j] = Vertex{
			DstX: float32(p.X),
			DstY: float32(p.Y),
			SrcX: float32(float64(srcBounds.Min.X) + (m11*dx-m01*dy)/det),
			SrcY: float32(float64(srcBounds.Min.Y) + (m00*dy-m10*dx)/det),
		}
	}

	var op DrawTrianglesShaderOptions
	op.Blend = BlendCopy
	op.Images[0] = offscreen
	dst.DrawTrianglesShader32(vertices[:], graphics.QuadIndices(), finalScreenShader(filter, opaque), &op)
}

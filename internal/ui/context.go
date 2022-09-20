// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

const screenShader = `package main

var Scale vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	sourceSize := imageSrcTextureSize()
	// texelSize is one pixel size in texel sizes.
	texelSize := 1 / sourceSize
	halfScaledTexelSize := texelSize / 2 / Scale

	// Shift 1/512 [texel] to avoid the tie-breaking issue.
	// As all the vertex positions are aligned to 1/16 [pixel], this shiting should work in most cases.
	pos := texCoord
	p0 := pos - halfScaledTexelSize + (texelSize / 512)
	p1 := pos + halfScaledTexelSize + (texelSize / 512)

	// Texels must be in the source rect, so it is not necessary to check.
	c0 := imageSrc0UnsafeAt(p0)
	c1 := imageSrc0UnsafeAt(vec2(p1.x, p0.y))
	c2 := imageSrc0UnsafeAt(vec2(p0.x, p1.y))
	c3 := imageSrc0UnsafeAt(p1)

	// p is the p1 value in one pixel assuming that the pixel's upper-left is (0, 0) and the lower-right is (1, 1).
	p := fract(p1 * sourceSize)

	// rate indicates how much the 4 colors are mixed. rate is in between [0, 1].
	//
	//     0 <= p <= 1/Scale: The rate is in between [0, 1]
	//     1/Scale < p:       Don't care. Adjacent colors (e.g. c0 vs c1 in an X direction) should be the same.
	rate := clamp(p*Scale, 0, 1)
	return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
}
`

type Game interface {
	NewOffscreenImage(width, height int) *Image
	Layout(outsideWidth, outsideHeight int) (int, int)
	Update() error
	Draw()
}

type context struct {
	game Game

	updateCalled bool

	offscreen *Image
	screen    *Image

	// The following members must be protected by the mutex m.
	outsideWidth  float64
	outsideHeight float64

	screenShader *Shader

	m sync.Mutex
}

func newContext(game Game) *context {
	return &context{
		game: game,
	}
}

func (c *context) updateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrameImpl(graphicsDriver, clock.UpdateFrame(), outsideWidth, outsideHeight, deviceScaleFactor)
}

func (c *context) forceUpdateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64) error {
	n := 1
	if graphicsDriver.IsDirectX() {
		// On DirectX, both framebuffers in the swap chain should be updated.
		// Or, the rendering result becomes unexpected when the window is resized.
		n = 2
	}
	for i := 0; i < n; i++ {
		if err := c.updateFrameImpl(graphicsDriver, 1, outsideWidth, outsideHeight, deviceScaleFactor); err != nil {
			return err
		}
	}
	return nil
}

func (c *context) updateFrameImpl(graphicsDriver graphicsdriver.Graphics, updateCount int, outsideWidth, outsideHeight float64, deviceScaleFactor float64) (err error) {
	if err := theGlobalState.error(); err != nil {
		return err
	}

	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return nil
	}

	debug.Logf("----\n")

	if err := buffered.BeginFrame(graphicsDriver); err != nil {
		return err
	}
	defer func() {
		// All the vertices data are consumed at the end of the frame, and the data backend can be
		// available after that. Until then, lock the vertices backend.
		err1 := graphics.LockAndResetVertices(func() error {
			if err := buffered.EndFrame(graphicsDriver); err != nil {
				return err
			}
			return nil
		})
		if err == nil {
			err = err1
		}
	}()

	// Create a shader for the screen if necessary.
	if c.screenShader == nil {
		ir, err := graphics.CompileShader([]byte(screenShader))
		if err != nil {
			return err
		}
		c.screenShader = NewShader(ir)
	}

	// ForceUpdate can be invoked even if the context is not initialized yet (#1591).
	if w, h := c.layoutGame(outsideWidth, outsideHeight, deviceScaleFactor); w == 0 || h == 0 {
		return nil
	}

	// Ensure that Update is called once before Draw so that Update can be used for initialization.
	if !c.updateCalled && updateCount == 0 {
		updateCount = 1
		c.updateCalled = true
	}
	debug.Logf("Update count per frame: %d\n", updateCount)

	// Update the game.
	for i := 0; i < updateCount; i++ {
		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.game.Update(); err != nil {
			return err
		}
		// Catch the error that happened at (*Image).At.
		if err := theGlobalState.error(); err != nil {
			return err
		}
		theUI.resetForTick()
	}

	// Draw the game.
	c.drawGame(graphicsDriver)

	// All the vertices data are consumed at the end of the frame, and the data backend can be
	// available after that. Until then, lock the vertices backend.
	return nil
}

func (c *context) drawGame(graphicsDriver graphicsdriver.Graphics) {
	if c.offscreen.volatile != theGlobalState.isScreenClearedEveryFrame() {
		w, h := c.offscreen.width, c.offscreen.height
		c.offscreen.MarkDisposed()
		c.offscreen = c.game.NewOffscreenImage(w, h)
	}

	// Even though updateCount == 0, the offscreen is cleared and Draw is called.
	// Draw should not update the game state and then the screen should not be updated without Update, but
	// users might want to process something at Draw with the time intervals of FPS.
	if theGlobalState.isScreenClearedEveryFrame() {
		c.offscreen.clear()
	}
	c.game.Draw()

	if graphicsDriver.NeedsClearingScreen() {
		// This clear is needed for fullscreen mode or some mobile platforms (#622).
		c.screen.clear()
	}

	ga := 1.0
	gd := 1.0
	gtx := 0.0
	gty := 0.0

	screenScale, offsetX, offsetY := c.screenScaleAndOffsets()
	s := screenScale
	switch y := graphicsDriver.FramebufferYDirection(); y {
	case graphicsdriver.Upward:
		ga *= s
		gd *= -s
		gty += float64(c.offscreen.height) * s
	case graphicsdriver.Downward:
		ga *= s
		gd *= s
	default:
		panic(fmt.Sprintf("ui: invalid y-direction: %d", y))
	}

	gtx += offsetX
	gty += offsetY

	var filter graphicsdriver.Filter
	var screenFilter bool
	switch {
	case !theGlobalState.isScreenFilterEnabled():
		filter = graphicsdriver.FilterNearest
	case math.Floor(s) == s:
		filter = graphicsdriver.FilterNearest
	case s > 1:
		screenFilter = true
	default:
		// screenShader works with >=1 scale, but does not well with <1 scale.
		// Use regular FilterLinear instead so far (#669).
		filter = graphicsdriver.FilterLinear
	}

	dstRegion := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(c.screen.width),
		Height: float32(c.screen.height),
	}

	vs := graphics.QuadVertices(
		0, 0, float32(c.offscreen.width), float32(c.offscreen.height),
		float32(ga), 0, 0, float32(gd), float32(gtx), float32(gty),
		1, 1, 1, 1)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderImageCount]*Image{c.offscreen}

	var shader *Shader
	var uniforms [][]float32
	if screenFilter {
		shader = c.screenShader
		dstWidth, dstHeight := c.screen.width, c.screen.height
		srcWidth, srcHeight := c.offscreen.width, c.offscreen.height
		uniforms = shader.ConvertUniforms(map[string]interface{}{
			"Scale": []float32{
				float32(dstWidth) / float32(srcWidth),
				float32(dstHeight) / float32(srcHeight),
			},
		})
	}
	c.screen.DrawTriangles(srcs, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, filter, graphicsdriver.AddressUnsafe, dstRegion, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, shader, uniforms, false, true)
}

func (c *context) layoutGame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
	c.m.Lock()
	defer c.m.Unlock()

	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight

	// Adjust the outside size to integer values.
	// Even if the original value is less than 1, the value must be a positive integer (#2340).
	iow, ioh := int(outsideWidth), int(outsideHeight)
	if iow == 0 {
		iow = 1
	}
	if ioh == 0 {
		ioh = 1
	}
	ow, oh := c.game.Layout(iow, ioh)
	if ow <= 0 || oh <= 0 {
		panic("ui: Layout must return positive numbers")
	}

	sw, sh := int(outsideWidth*deviceScaleFactor), int(outsideHeight*deviceScaleFactor)
	if c.screen != nil {
		if c.screen.width != sw || c.screen.height != sh {
			c.screen.MarkDisposed()
			c.screen = nil
		}
	}
	if c.screen == nil {
		c.screen = NewImage(sw, sh, atlas.ImageTypeScreen)
	}

	if c.offscreen != nil {
		if c.offscreen.width != ow || c.offscreen.height != oh {
			c.offscreen.MarkDisposed()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = c.game.NewOffscreenImage(ow, oh)
	}

	return ow, oh
}

func (c *context) adjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets()
	// The scale 0 indicates that the screen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

func (c *context) screenScaleAndOffsets() (float64, float64, float64) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.screen == nil {
		return 0, 0, 0
	}

	scaleX := float64(c.screen.width) / float64(c.offscreen.width)
	scaleY := float64(c.screen.height) / float64(c.offscreen.height)
	scale := math.Min(scaleX, scaleY)
	width := float64(c.offscreen.width) * scale
	height := float64(c.offscreen.height) * scale
	x := (float64(c.screen.width) - width) / 2
	y := (float64(c.screen.height) - height) / 2
	return scale, x, y
}

var theGlobalState = globalState{
	isScreenClearedEveryFrame_: 1,
	screenFilterEnabled_:       1,
}

// globalState represents a global state in this package.
// This is available even before the game loop starts.
type globalState struct {
	err_ error
	errM sync.Mutex

	fpsMode_                   int32
	isScreenClearedEveryFrame_ int32
	screenFilterEnabled_       int32
	graphicsLibrary_           int32
}

func (g *globalState) error() error {
	g.errM.Lock()
	defer g.errM.Unlock()
	return g.err_
}

func (g *globalState) setError(err error) {
	g.errM.Lock()
	defer g.errM.Unlock()
	if g.err_ == nil {
		g.err_ = err
	}
}

func (g *globalState) fpsMode() FPSModeType {
	return FPSModeType(atomic.LoadInt32(&g.fpsMode_))
}

func (g *globalState) setFPSMode(fpsMode FPSModeType) {
	atomic.StoreInt32(&g.fpsMode_, int32(fpsMode))
}

func (g *globalState) isScreenClearedEveryFrame() bool {
	return atomic.LoadInt32(&g.isScreenClearedEveryFrame_) != 0
}

func (g *globalState) setScreenClearedEveryFrame(cleared bool) {
	v := int32(0)
	if cleared {
		v = 1
	}
	atomic.StoreInt32(&g.isScreenClearedEveryFrame_, v)
}

func (g *globalState) isScreenFilterEnabled() bool {
	return graphicsdriver.Filter(atomic.LoadInt32(&g.screenFilterEnabled_)) != 0
}

func (g *globalState) setScreenFilterEnabled(enabled bool) {
	v := int32(0)
	if enabled {
		v = 1
	}
	atomic.StoreInt32(&g.screenFilterEnabled_, v)
}

func (g *globalState) setGraphicsLibrary(library GraphicsLibrary) {
	atomic.StoreInt32(&g.graphicsLibrary_, int32(library))
}

func (g *globalState) graphicsLibrary() GraphicsLibrary {
	return GraphicsLibrary(atomic.LoadInt32(&g.graphicsLibrary_))
}

func FPSMode() FPSModeType {
	return theGlobalState.fpsMode()
}

func SetFPSMode(fpsMode FPSModeType) {
	theGlobalState.setFPSMode(fpsMode)
	theUI.SetFPSMode(fpsMode)
}

func IsScreenClearedEveryFrame() bool {
	return theGlobalState.isScreenClearedEveryFrame()
}

func SetScreenClearedEveryFrame(cleared bool) {
	theGlobalState.setScreenClearedEveryFrame(cleared)
}

func IsScreenFilterEnabled() bool {
	return theGlobalState.isScreenFilterEnabled()
}

func SetScreenFilterEnabled(enabled bool) {
	theGlobalState.setScreenFilterEnabled(enabled)
}

func GetGraphicsLibrary() GraphicsLibrary {
	return theGlobalState.graphicsLibrary()
}

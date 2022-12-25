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
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

var (
	NearestFilterShader = &Shader{shader: mipmap.NearestFilterShader}
	LinearFilterShader  = &Shader{shader: mipmap.LinearFilterShader}
)

type Game interface {
	NewOffscreenImage(width, height int) *Image
	NewScreenImage(width, height int) *Image
	Layout(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64)
	UpdateInputState(fn func(*InputState))
	Update() error
	DrawOffscreen() error
	DrawFinalScreen(scale, offsetX, offsetY float64)
}

type context struct {
	game Game

	updateCalled bool

	offscreen *Image
	screen    *Image

	screenWidth     float64
	screenHeight    float64
	offscreenWidth  float64
	offscreenHeight float64

	isOffscreenModified bool

	skipCount int

	setContextOnce sync.Once
}

func newContext(game Game) *context {
	return &context{
		game: game,
	}
}

func (c *context) updateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrameImpl(graphicsDriver, clock.UpdateFrame(), outsideWidth, outsideHeight, deviceScaleFactor, ui, false)
}

func (c *context) forceUpdateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl) error {
	n := 1
	if graphicsDriver.IsDirectX() {
		// On DirectX, both framebuffers in the swap chain should be updated.
		// Or, the rendering result becomes unexpected when the window is resized.
		n = 2
	}
	for i := 0; i < n; i++ {
		if err := c.updateFrameImpl(graphicsDriver, 1, outsideWidth, outsideHeight, deviceScaleFactor, ui, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *context) updateFrameImpl(graphicsDriver graphicsdriver.Graphics, updateCount int, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl, forceDraw bool) (err error) {
	if err := theGlobalState.error(); err != nil {
		return err
	}

	ui.beginFrame()
	defer ui.endFrame()

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
		if err1 := buffered.EndFrame(graphicsDriver); err == nil && err1 != nil {
			err = err1
		}
	}()

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
		// Read the input state and use it for one tick to give a consistent result for one tick (#2496, #2501).
		c.game.UpdateInputState(func(inputState *InputState) {
			ui.readInputState(inputState)
		})

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
	}

	// Update window icons during a frame, since an icon might be *ebiten.Image and
	// getting pixels from it needs to be in a frame (#1468).
	if err := ui.updateIconIfNeeded(); err != nil {
		return err
	}

	// Draw the game.
	if err := c.drawGame(graphicsDriver, forceDraw); err != nil {
		return err
	}

	return nil
}

func (c *context) newOffscreenImage(w, h int) *Image {
	img := c.game.NewOffscreenImage(w, h)
	img.modifyCallback = func() {
		c.isOffscreenModified = true
	}
	return img
}

func (c *context) drawGame(graphicsDriver graphicsdriver.Graphics, forceDraw bool) error {
	if (c.offscreen.imageType == atlas.ImageTypeVolatile) != theGlobalState.isScreenClearedEveryFrame() {
		w, h := c.offscreen.width, c.offscreen.height
		c.offscreen.MarkDisposed()
		c.offscreen = c.newOffscreenImage(w, h)
	}

	// isOffscreenModified is updated when an offscreen's modifyCallback.
	c.isOffscreenModified = false

	// Even though updateCount == 0, the offscreen is cleared and Draw is called.
	// Draw should not update the game state and then the screen should not be updated without Update, but
	// users might want to process something at Draw with the time intervals of FPS.
	if theGlobalState.isScreenClearedEveryFrame() {
		c.offscreen.clear()
	}

	if err := c.game.DrawOffscreen(); err != nil {
		return err
	}

	const maxSkipCount = 3

	if !forceDraw && !c.isOffscreenModified {
		if c.skipCount < maxSkipCount {
			c.skipCount++
		}
	} else {
		c.skipCount = 0
	}

	if c.skipCount < maxSkipCount {
		if graphicsDriver.NeedsClearingScreen() {
			// This clear is needed for fullscreen mode or some mobile platforms (#622).
			c.screen.clear()
		}

		c.game.DrawFinalScreen(c.screenScaleAndOffsets())

		// The final screen is never used as the rendering source.
		// Flush its buffer here just in case.
		c.screen.flushBufferIfNeeded()
	}

	return nil
}

func (c *context) layoutGame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
	owf, ohf := c.game.Layout(outsideWidth, outsideHeight)
	if owf <= 0 || ohf <= 0 {
		panic("ui: Layout must return positive numbers")
	}

	c.screenWidth = outsideWidth * deviceScaleFactor
	c.screenHeight = outsideHeight * deviceScaleFactor
	c.offscreenWidth = owf
	c.offscreenHeight = ohf

	sw := int(math.Ceil(c.screenWidth))
	sh := int(math.Ceil(c.screenHeight))
	ow := int(math.Ceil(c.offscreenWidth))
	oh := int(math.Ceil(c.offscreenHeight))

	if c.screen != nil {
		if c.screen.width != sw || c.screen.height != sh {
			c.screen.MarkDisposed()
			c.screen = nil
		}
	}
	if c.screen == nil {
		c.screen = c.game.NewScreenImage(sw, sh)
	}

	if c.offscreen != nil {
		if c.offscreen.width != ow || c.offscreen.height != oh {
			c.offscreen.MarkDisposed()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = c.newOffscreenImage(ow, oh)
	}

	return ow, oh
}

func (c *context) clientPositionToLogicalPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets()
	// The scale 0 indicates that the screen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

func (c *context) logicalPositionToClientPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets()
	return (x*s + ox) / deviceScaleFactor, (y*s + oy) / deviceScaleFactor
}

func (c *context) screenScaleAndOffsets() (scale, offsetX, offsetY float64) {
	scaleX := c.screenWidth / c.offscreenWidth
	scaleY := c.screenHeight / c.offscreenHeight
	scale = math.Min(scaleX, scaleY)
	width := c.offscreenWidth * scale
	height := c.offscreenHeight * scale
	offsetX = (c.screenWidth - width) / 2
	offsetY = (c.screenHeight - height) / 2
	return
}

func LogicalPositionToClientPosition(x, y float64) (float64, float64) {
	return theUI.context.logicalPositionToClientPosition(x, y, theUI.DeviceScaleFactor())
}

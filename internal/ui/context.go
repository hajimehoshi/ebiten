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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
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
	Layout(outsideWidth, outsideHeight int) (int, int)
	Update() error
	DrawOffscreen() error
	DrawScreen()
	ScreenScaleAndOffsets() (scale, offsetX, offsetY float64)
}

type context struct {
	game Game

	updateCalled bool

	offscreen *Image
	screen    *Image

	// The following members must be protected by the mutex m.
	outsideWidth  float64
	outsideHeight float64
}

func newContext(game Game) *context {
	return &context{
		game: game,
	}
}

func (c *context) updateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrameImpl(graphicsDriver, clock.UpdateFrame(), outsideWidth, outsideHeight, deviceScaleFactor, ui)
}

func (c *context) forceUpdateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl) error {
	n := 1
	if graphicsDriver.IsDirectX() {
		// On DirectX, both framebuffers in the swap chain should be updated.
		// Or, the rendering result becomes unexpected when the window is resized.
		n = 2
	}
	for i := 0; i < n; i++ {
		if err := c.updateFrameImpl(graphicsDriver, 1, outsideWidth, outsideHeight, deviceScaleFactor, ui); err != nil {
			return err
		}
	}
	return nil
}

func (c *context) updateFrameImpl(graphicsDriver graphicsdriver.Graphics, updateCount int, outsideWidth, outsideHeight float64, deviceScaleFactor float64, ui *userInterfaceImpl) (err error) {
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
		ui.resetForTick()
	}

	// Draw the game.
	if err := c.drawGame(graphicsDriver); err != nil {
		return err
	}

	// All the vertices data are consumed at the end of the frame, and the data backend can be
	// available after that. Until then, lock the vertices backend.
	return nil
}

func (c *context) drawGame(graphicsDriver graphicsdriver.Graphics) error {
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
	if err := c.game.DrawOffscreen(); err != nil {
		return err
	}

	if graphicsDriver.NeedsClearingScreen() {
		// This clear is needed for fullscreen mode or some mobile platforms (#622).
		c.screen.clear()
	}

	c.game.DrawScreen()

	// The final screen is never used as the rendering source.
	// Flush its buffer here just in case.
	c.screen.flushBufferIfNeeded()

	return nil
}

func (c *context) layoutGame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
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
		c.screen = c.game.NewScreenImage(sw, sh)
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
	s, ox, oy := c.game.ScreenScaleAndOffsets()
	// The scale 0 indicates that the screen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

var theGlobalState = globalState{
	isScreenClearedEveryFrame_: 1,
}

// globalState represents a global state in this package.
// This is available even before the game loop starts.
type globalState struct {
	err_ error
	errM sync.Mutex

	fpsMode_                   int32
	isScreenClearedEveryFrame_ int32
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

func GetGraphicsLibrary() GraphicsLibrary {
	return theGlobalState.graphicsLibrary()
}

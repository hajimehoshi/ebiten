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
	graphicspkg "github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

const DefaultTPS = 60

type Game interface {
	Layout(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int)
	Update() error
	Draw(screenScale float64, offsetX, offsetY float64, needsClearingScreen bool, framebufferYDirection graphicsdriver.YDirection, screenClearedEveryFrame, filterEnabled bool) error
}

type contextImpl struct {
	game Game

	updateCalled bool

	// The following members must be protected by the mutex m.
	outsideWidth  float64
	outsideHeight float64
	screenWidth   int
	screenHeight  int

	m sync.Mutex
}

func newContextImpl(game Game) *contextImpl {
	return &contextImpl{
		game: game,
	}
}

func (c *contextImpl) updateFrame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrameImpl(clock.Update(theGlobalState.maxTPS()), outsideWidth, outsideHeight, deviceScaleFactor)
}

func (c *contextImpl) forceUpdateFrame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) error {
	return c.updateFrameImpl(1, outsideWidth, outsideHeight, deviceScaleFactor)
}

func (c *contextImpl) updateFrameImpl(updateCount int, outsideWidth, outsideHeight float64, deviceScaleFactor float64) error {
	if err := theGlobalState.err(); err != nil {
		return err
	}

	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return nil
	}

	// ForceUpdate can be invoked even if the context is not initialized yet (#1591).
	if w, h := c.layoutGame(outsideWidth, outsideHeight, deviceScaleFactor); w == 0 || h == 0 {
		return nil
	}

	debug.Logf("----\n")

	if err := buffered.BeginFrame(); err != nil {
		return err
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
		Get().resetForTick()
	}

	// Draw the game.
	screenScale, offsetX, offsetY := c.screenScaleAndOffsets(deviceScaleFactor)
	if err := c.game.Draw(screenScale, offsetX, offsetY, graphics().NeedsClearingScreen(), graphics().FramebufferYDirection(), theGlobalState.isScreenClearedEveryFrame(), theGlobalState.isScreenFilterEnabled()); err != nil {
		return err
	}

	// All the vertices data are consumed at the end of the frame, and the data backend can be
	// available after that. Until then, lock the vertices backend.
	return graphicspkg.LockAndResetVertices(func() error {
		if err := buffered.EndFrame(); err != nil {
			return err
		}
		return nil
	})
}

func (c *contextImpl) layoutGame(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
	c.m.Lock()
	defer c.m.Unlock()

	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
	w, h := c.game.Layout(outsideWidth, outsideHeight, deviceScaleFactor)
	c.screenWidth = w
	c.screenHeight = h
	return w, h
}

func (c *contextImpl) adjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets(deviceScaleFactor)
	// The scale 0 indicates that the screen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

func (c *contextImpl) screenScaleAndOffsets(deviceScaleFactor float64) (float64, float64, float64) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.screenWidth == 0 || c.screenHeight == 0 {
		return 0, 0, 0
	}

	scaleX := c.outsideWidth / float64(c.screenWidth) * deviceScaleFactor
	scaleY := c.outsideHeight / float64(c.screenHeight) * deviceScaleFactor
	scale := math.Min(scaleX, scaleY)
	width := float64(c.screenWidth) * scale
	height := float64(c.screenHeight) * scale
	x := (c.outsideWidth*deviceScaleFactor - width) / 2
	y := (c.outsideHeight*deviceScaleFactor - height) / 2
	return scale, x, y
}

var theGlobalState = globalState{
	maxTPS_:                    DefaultTPS,
	isScreenClearedEveryFrame_: 1,
	screenFilterEnabled_:       1,
}

// globalState represents a global state in this package.
// This is available even before the game loop starts.
type globalState struct {
	err_                       atomic.Value
	fpsMode_                   int32
	maxTPS_                    int32
	isScreenClearedEveryFrame_ int32
	screenFilterEnabled_       int32
}

func (g *globalState) err() error {
	err, ok := g.err_.Load().(error)
	if !ok {
		return nil
	}
	return err
}

func (g *globalState) setError(err error) {
	g.err_.Store(err)
}

func (g *globalState) fpsMode() FPSModeType {
	return FPSModeType(atomic.LoadInt32(&g.fpsMode_))
}

func (g *globalState) setFPSMode(fpsMode FPSModeType) {
	atomic.StoreInt32(&g.fpsMode_, int32(fpsMode))
}

func (g *globalState) maxTPS() int {
	if g.fpsMode() == FPSModeVsyncOffMinimum {
		return clock.SyncWithFPS
	}
	return int(atomic.LoadInt32(&g.maxTPS_))
}

func (g *globalState) setMaxTPS(tps int) {
	if tps < 0 && tps != clock.SyncWithFPS {
		panic("ebiten: tps must be >= 0 or SyncWithFPS")
	}
	atomic.StoreInt32(&g.maxTPS_, int32(tps))
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

func SetError(err error) {
	theGlobalState.setError(err)
}

func FPSMode() FPSModeType {
	return theGlobalState.fpsMode()
}

func SetFPSMode(fpsMode FPSModeType) {
	theGlobalState.setFPSMode(fpsMode)
	Get().SetFPSMode(fpsMode)
}

func MaxTPS() int {
	return theGlobalState.maxTPS()
}

func SetMaxTPS(tps int) {
	theGlobalState.setMaxTPS(tps)
}

func IsScreenClearedEveryFrame() bool {
	return theGlobalState.isScreenClearedEveryFrame()
}

func SetScreenClearedEveryFrame(cleared bool) {
	theGlobalState.setScreenClearedEveryFrame(cleared)
}

func SetScreenFilterEnabled(enabled bool) {
	theGlobalState.setScreenFilterEnabled(enabled)
}

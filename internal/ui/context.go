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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	graphicspkg "github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

const DefaultTPS = 60

type Context interface {
	UpdateOffscreen(outsideWidth, outsideHeight float64) (int, int)
	UpdateFrame(updateCount int, screenScale float64, offsetX, offsetY float64) error
}

type contextImpl struct {
	context Context

	outsideWidth    float64
	outsideHeight   float64
	offscreenWidth  int
	offscreenHeight int
}

func newContextImpl(context Context) *contextImpl {
	return &contextImpl{
		context: context,
	}
}

func (c *contextImpl) updateFrame(deviceScaleFactor float64) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrameImpl(clock.Update(theGlobalState.maxTPS()), deviceScaleFactor)
}

func (c *contextImpl) forceUpdateFrame(deviceScaleFactor float64) error {
	return c.updateFrameImpl(1, deviceScaleFactor)
}

func (c *contextImpl) updateFrameImpl(updateCount int, deviceScaleFactor float64) error {
	ow, oh := c.context.UpdateOffscreen(c.outsideWidth, c.outsideHeight)
	c.offscreenWidth = ow
	c.offscreenHeight = oh

	if err := theGlobalState.err(); err != nil {
		return err
	}

	// ForceUpdate can be invoked even if the context is not initialized yet (#1591).
	if c.outsideWidth == 0 || c.outsideHeight == 0 {
		return nil
	}

	debug.Logf("----\n")

	if err := buffered.BeginFrame(); err != nil {
		return err
	}

	screenScale := c.screenScale(deviceScaleFactor)
	offsetX, offsetY := c.offsets(deviceScaleFactor)
	if err := c.context.UpdateFrame(updateCount, screenScale, offsetX, offsetY); err != nil {
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

func (c *contextImpl) layout(outsideWidth, outsideHeight float64) {
	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return
	}

	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
}

// TODO: Make adjustPosition concurrent-safe.
func (c *contextImpl) adjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	ox, oy := c.offsets(deviceScaleFactor)
	s := c.screenScale(deviceScaleFactor)
	// The scale 0 indicates that the offscreen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

func (c *contextImpl) offsets(deviceScaleFactor float64) (float64, float64) {
	if c.offscreenWidth == 0 || c.offscreenHeight == 0 {
		return 0, 0
	}
	s := c.screenScale(deviceScaleFactor)
	width := float64(c.offscreenWidth) * s
	height := float64(c.offscreenHeight) * s
	x := (c.outsideWidth*deviceScaleFactor - width) / 2
	y := (c.outsideHeight*deviceScaleFactor - height) / 2
	return x, y
}

func (c *contextImpl) screenScale(deviceScaleFactor float64) float64 {
	if c.offscreenWidth == 0 || c.offscreenHeight == 0 {
		return 0
	}
	scaleX := c.outsideWidth / float64(c.offscreenWidth) * deviceScaleFactor
	scaleY := c.outsideHeight / float64(c.offscreenHeight) * deviceScaleFactor
	return math.Min(scaleX, scaleY)
}

var theGlobalState = globalState{
	currentMaxTPS: DefaultTPS,
}

// globalState represents a global state in this package.
// This is available even before the game loop starts.
type globalState struct {
	currentErr     atomic.Value
	currentFPSMode int32
	currentMaxTPS  int32
}

func (g *globalState) err() error {
	err, ok := g.currentErr.Load().(error)
	if !ok {
		return nil
	}
	return err
}

func (g *globalState) setError(err error) {
	g.currentErr.Store(err)
}

func (g *globalState) fpsMode() FPSModeType {
	return FPSModeType(atomic.LoadInt32(&g.currentFPSMode))
}

func (g *globalState) setFPSMode(fpsMode FPSModeType) {
	atomic.StoreInt32(&g.currentFPSMode, int32(fpsMode))
}

func (g *globalState) maxTPS() int {
	if g.fpsMode() == FPSModeVsyncOffMinimum {
		return clock.SyncWithFPS
	}
	return int(atomic.LoadInt32(&g.currentMaxTPS))
}

func (g *globalState) setMaxTPS(tps int) {
	if tps < 0 && tps != clock.SyncWithFPS {
		panic("ebiten: tps must be >= 0 or SyncWithFPS")
	}
	atomic.StoreInt32(&g.currentMaxTPS, int32(tps))
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

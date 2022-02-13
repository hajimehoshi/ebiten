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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
)

const DefaultTPS = 60

type Context interface {
	UpdateFrame(updateCount int) error
	Layout(outsideWidth, outsideHeight float64)

	// AdjustPosition can be called from a different goroutine from Update's or Layout's.
	AdjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64)
}

type contextImpl struct {
	context Context

	outsideWidth  float64
	outsideHeight float64
}

func newContextImpl(context Context) *contextImpl {
	return &contextImpl{
		context: context,
	}
}

func (c *contextImpl) updateFrame() error {
	if err := theGlobalState.err(); err != nil {
		return err
	}

	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.context.UpdateFrame(clock.Update(theGlobalState.maxTPS()))
}

func (c *contextImpl) forceUpdateFrame() error {
	if err := theGlobalState.err(); err != nil {
		return err
	}

	// ForceUpdate can be invoked even if the context is not initialized yet (#1591).
	if c.outsideWidth == 0 || c.outsideHeight == 0 {
		return nil
	}

	return c.context.UpdateFrame(1)
}

func (c *contextImpl) layout(outsideWidth, outsideHeight float64) {
	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return
	}

	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
	c.context.Layout(outsideWidth, outsideHeight)
}

func (c *contextImpl) adjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	return c.context.AdjustPosition(x, y, deviceScaleFactor)
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

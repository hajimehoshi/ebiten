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
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

type uiContext struct {
	game      Game
	offscreen *Image
	screen    *Image

	updateCalled bool

	// scaleForWindow is the scale of a window. This doesn't represent the scale on fullscreen. This value works
	// only on desktops.
	//
	// scaleForWindow is for backward compatibility and is used to calculate the window size when SetScreenSize
	// is called.
	scaleForWindow float64

	outsideSizeUpdated bool
	outsideWidth       float64
	outsideHeight      float64

	err atomic.Value

	m sync.Mutex
}

var theUIContext = &uiContext{}

func (c *uiContext) set(game Game, scaleForWindow float64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.game = game
}

func (c *uiContext) setError(err error) {
	c.err.Store(err)
}

func (c *uiContext) Layout(outsideWidth, outsideHeight float64) {
	c.outsideSizeUpdated = true
	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
}

func (c *uiContext) updateOffscreen() {
	sw, sh := c.game.Layout(int(c.outsideWidth), int(c.outsideHeight))
	if sw <= 0 || sh <= 0 {
		panic("ebiten: Layout must return positive numbers")
	}

	if c.offscreen != nil && !c.outsideSizeUpdated {
		if w, h := c.offscreen.Size(); w == sw && h == sh {
			return
		}
	}
	c.outsideSizeUpdated = false

	if c.screen != nil {
		c.screen.Dispose()
		c.screen = nil
	}

	if c.offscreen != nil {
		if w, h := c.offscreen.Size(); w != sw || h != sh {
			c.offscreen.Dispose()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = NewImage(sw, sh)
		c.offscreen.mipmap.SetVolatile(IsScreenClearedEveryFrame())
	}

	// TODO: This is duplicated with mobile/ebitenmobileview/funcs.go. Refactor this.
	d := uiDriver().DeviceScaleFactor()
	c.screen = newScreenFramebufferImage(int(c.outsideWidth*d), int(c.outsideHeight*d))

	// Do not have to update scaleForWindow since this is used only for backward compatibility.
	// Then, if a window is resizable, scaleForWindow (= ebiten.ScreenScale) might not match with the actual
	// scale. This is fine since ebiten.ScreenScale will be deprecated.
}

func (c *uiContext) setScreenClearedEveryFrame(cleared bool) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.offscreen != nil {
		c.offscreen.mipmap.SetVolatile(cleared)
	}
}

func (c *uiContext) setWindowResizable(resizable bool) {
	c.m.Lock()
	defer c.m.Unlock()

	if w := uiDriver().Window(); w != nil {
		w.SetResizable(resizable)
	}
}

func (c *uiContext) screenScale() float64 {
	if c.offscreen == nil {
		return 0
	}
	sw, sh := c.offscreen.Size()
	d := uiDriver().DeviceScaleFactor()
	scaleX := c.outsideWidth / float64(sw) * d
	scaleY := c.outsideHeight / float64(sh) * d
	return math.Min(scaleX, scaleY)
}

func (c *uiContext) offsets() (float64, float64) {
	if c.offscreen == nil {
		return 0, 0
	}
	sw, sh := c.offscreen.Size()
	d := uiDriver().DeviceScaleFactor()
	s := c.screenScale()
	width := float64(sw) * s
	height := float64(sh) * s
	return (c.outsideWidth*d - width) / 2, (c.outsideHeight*d - height) / 2
}

func (c *uiContext) Update() error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.

	if err, ok := c.err.Load().(error); ok && err != nil {
		return err
	}
	if err := buffered.BeginFrame(); err != nil {
		return err
	}
	if err := c.update(); err != nil {
		return err
	}
	if err := buffered.EndFrame(); err != nil {
		return err
	}
	return nil
}

func (c *uiContext) Draw() error {
	if err, ok := c.err.Load().(error); ok && err != nil {
		return err
	}
	if err := buffered.BeginFrame(); err != nil {
		return err
	}
	c.draw()
	if err := buffered.EndFrame(); err != nil {
		return err
	}
	return nil
}

func (c *uiContext) update() error {
	// TODO: Move the clock usage to the UI driver side.
	updateCount := clock.Update(MaxTPS())

	// Ensure that Update is called once before Draw so that Update can be used for initialization.
	if !c.updateCalled && updateCount == 0 {
		updateCount = 1
		c.updateCalled = true
	}

	for i := 0; i < updateCount; i++ {
		c.updateOffscreen()

		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.game.Update(); err != nil {
			return err
		}
		uiDriver().ResetForFrame()
	}
	return nil
}

func (c *uiContext) draw() {
	// c.screen might be nil when updateCount is 0 in the initial state (#1039).
	if c.screen == nil {
		return
	}

	if IsScreenClearedEveryFrame() {
		c.offscreen.Clear()
	}
	c.game.Draw(c.offscreen)

	// This clear is needed for fullscreen mode or some mobile platforms (#622).
	c.screen.Clear()

	op := &DrawImageOptions{}

	s := c.screenScale()
	switch vd := uiDriver().Graphics().FramebufferYDirection(); vd {
	case driver.Upward:
		op.GeoM.Scale(s, -s)
		_, h := c.offscreen.Size()
		op.GeoM.Translate(0, float64(h)*s)
	case driver.Downward:
		op.GeoM.Scale(s, s)
	default:
		panic(fmt.Sprintf("ebiten: invalid v-direction: %d", vd))
	}

	op.GeoM.Translate(c.offsets())
	op.CompositeMode = CompositeModeCopy

	// filterScreen works with >=1 scale, but does not well with <1 scale.
	// Use regular FilterLinear instead so far (#669).
	if s >= 1 {
		op.Filter = filterScreen
	} else {
		op.Filter = FilterLinear
	}
	c.screen.DrawImage(c.offscreen, op)
}

func (c *uiContext) AdjustPosition(x, y float64) (float64, float64) {
	d := uiDriver().DeviceScaleFactor()
	ox, oy := c.offsets()
	s := c.screenScale()
	return (x*d - ox) / s, (y*d - oy) / s
}

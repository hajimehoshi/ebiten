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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

func newGraphicsContext(f func(*Image) error) *graphicsContext {
	return &graphicsContext{
		f: f,
	}
}

type graphicsContext struct {
	f                   func(*Image) error
	screen              *Image
	defaultRenderTarget *Image
	screenScale         int
	imageTasksDone      bool
}

func (c *graphicsContext) SetSize(screenWidth, screenHeight, screenScale int) error {
	if c.defaultRenderTarget != nil {
		c.defaultRenderTarget.Dispose()
	}
	if c.screen != nil {
		c.screen.Dispose()
	}
	screen, err := NewImage(screenWidth, screenHeight, FilterNearest)
	if err != nil {
		return err
	}
	c.defaultRenderTarget, err = newImageWithZeroFramebuffer(screenWidth*screenScale, screenHeight*screenScale)
	if err != nil {
		return err
	}
	c.defaultRenderTarget.Clear()
	c.screen = screen
	c.screenScale = screenScale
	return nil
}

func (c *graphicsContext) Update() error {
	if !c.imageTasksDone {
		if err := graphics.Initialize(ui.GLContext()); err != nil {
			return err
		}
		// This execution is called here because we can say actual GL function calls
		// should be done here (especailly on mobiles).
		if err := theDelayedImageTasks.exec(); err != nil {
			return err
		}
		c.imageTasksDone = true
	}
	if err := c.screen.Clear(); err != nil {
		return err
	}
	if err := c.f(c.screen); err != nil {
		return err
	}
	if IsRunningSlowly() {
		return nil
	}
	if err := c.defaultRenderTarget.Clear(); err != nil {
		return err
	}
	scale := float64(c.screenScale)
	options := &DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	if err := c.defaultRenderTarget.DrawImage(c.screen, options); err != nil {
		return err
	}
	// Call glFlush to prevent black flicking (especially on Android (#226)).
	ui.GLContext().Flush()
	return nil
}

func (c *graphicsContext) Resume() error {
	ui.GLContext().Resume()
	if !c.imageTasksDone {
		return nil
	}
	if err := graphics.Initialize(ui.GLContext()); err != nil {
		return err
	}
	if err := theImages.restorePixels(); err != nil {
		return err
	}
	return nil
}

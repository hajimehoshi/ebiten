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
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

func useGLContext(f func(*opengl.Context)) {
	ui.ExecOnUIThread(func() {
		f(glContext)
	})
}

func newGraphicsContext(screenWidth, screenHeight, screenScale int) (*graphicsContext, error) {
	c := &graphicsContext{}
	if err := c.setSize(screenWidth, screenHeight, screenScale); err != nil {
		return nil, err
	}
	return c, nil
}

type graphicsContext struct {
	screen              *Image
	defaultRenderTarget *Image
	screenScale         int
}

func (c *graphicsContext) preUpdate() error {
	return c.screen.Clear()
}

func (c *graphicsContext) postUpdate() error {
	// TODO: In WebGL, we don't need to clear the image here.
	if err := c.defaultRenderTarget.Clear(); err != nil {
		return err
	}

	scale := float64(c.screenScale)
	options := &DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	if err := c.defaultRenderTarget.DrawImage(c.screen, options); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) setSize(screenWidth, screenHeight, screenScale int) error {
	if c.defaultRenderTarget != nil {
		c.defaultRenderTarget.dispose()
	}
	if c.screen != nil {
		c.screen.dispose()
	}

	var err error
	useGLContext(func(g *opengl.Context) {
		f, err := graphics.NewZeroFramebuffer(g, screenWidth*screenScale, screenHeight*screenScale)
		if err != nil {
			return
		}

		texture, err := graphics.NewTexture(g, screenWidth, screenHeight, g.Nearest)
		if err != nil {
			return
		}
		screenF, err := graphics.NewFramebufferFromTexture(g, texture)
		if err != nil {
			return
		}
		screen := &Image{framebuffer: screenF, texture: texture}
		c.defaultRenderTarget = &Image{framebuffer: f, texture: nil}
		c.screen = screen
		c.screenScale = screenScale
	})
	return err
}

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
)

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

func (c *graphicsContext) update(f func(*Image) error) error {
	if err := c.screen.Clear(); err != nil {
		return err
	}
	if err := f(c.screen); err != nil {
		return err
	}
	if IsRunningSlowly() {
		return nil
	}
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
		c.defaultRenderTarget.Dispose()
	}
	if c.screen != nil {
		c.screen.Dispose()
	}

	f, err := graphics.NewZeroFramebuffer(glContext, screenWidth*screenScale, screenHeight*screenScale)
	if err != nil {
		return err
	}

	texture, err := graphics.NewTexture(glContext, screenWidth, screenHeight, glContext.Nearest)
	if err != nil {
		return err
	}
	screenF, err := graphics.NewFramebufferFromTexture(glContext, texture)
	if err != nil {
		return err
	}
	w, h := screenF.Size()
	screen := &Image{
		framebuffer: screenF,
		texture:     texture,
		width:       w,
		height:      h,
	}
	w, h = f.Size()
	c.defaultRenderTarget = &Image{
		framebuffer: f,
		texture:     nil,
		width:       w,
		height:      h,
	}
	c.defaultRenderTarget.Clear()
	c.screen = screen
	c.screenScale = screenScale
	return nil
}

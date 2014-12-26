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
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func newGraphicsContext(screenWidth, screenHeight, screenScale int) (*graphicsContext, error) {
	f, err := opengl.NewZeroFramebuffer(screenWidth*screenScale, screenHeight*screenScale)
	if err != nil {
		return nil, err
	}

	texture, err := opengl.NewTexture(screenWidth, screenHeight, opengl.Filter(FilterNearest))
	if err != nil {
		return nil, err
	}
	screen, err := newInnerImage(texture)
	if err != nil {
		return nil, err
	}

	c := &graphicsContext{
		defaultR:    &innerImage{f, nil},
		screen:      screen,
		screenScale: screenScale,
	}
	return c, nil
}

type graphicsContext struct {
	screen      *innerImage
	defaultR    *innerImage
	screenScale int
}

func (c *graphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	framebuffer := c.screen.framebuffer
	texture := c.screen.texture

	framebuffer.Dispose()
	texture.Dispose()
}

func (c *graphicsContext) preUpdate() error {
	return c.screen.Clear()
}

func (c *graphicsContext) postUpdate() error {
	if err := c.defaultR.Clear(); err != nil {
		return err
	}

	scale := float64(c.screenScale)
	options := &DrawImageOptions{
		GeometryMatrix: ScaleGeometry(scale, scale),
	}
	if err := c.defaultR.drawImage(c.screen, options); err != nil {
		return err
	}

	opengl.Flush()
	return nil
}

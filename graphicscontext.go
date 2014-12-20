/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func newGraphicsContext(screenWidth, screenHeight, screenScale int) (*graphicsContext, error) {
	r, err := opengl.NewZeroRenderTarget(screenWidth*screenScale, screenHeight*screenScale)
	if err != nil {
		return nil, err
	}

	screen, err := newRenderTarget(screenWidth, screenHeight, gl.NEAREST)
	if err != nil {
		return nil, err
	}
	c := &graphicsContext{
		defaultR:    &renderTarget{r, nil},
		screen:      screen,
		screenScale: screenScale,
	}
	return c, nil
}

type graphicsContext struct {
	screen      *renderTarget
	defaultR    *renderTarget
	screenScale int
}

func (c *graphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	glRenderTarget := c.screen.glRenderTarget
	texture := c.screen.texture
	glTexture := texture.glTexture

	glRenderTarget.Dispose()
	glTexture.Dispose()
}

func (c *graphicsContext) preUpdate() error {
	return c.screen.Clear()
}

func (c *graphicsContext) postUpdate() error {
	// We don't need to clear the default render target (framebuffer).
	// For the default framebuffer, a special shader is used.
	scale := float64(c.screenScale)
	geo := ScaleGeometry(scale, scale)
	clr := ColorMatrixI()
	if err := DrawWholeTexture(c.defaultR, c.screen.texture, geo, clr); err != nil {
		return err
	}

	gl.Flush()
	return nil
}

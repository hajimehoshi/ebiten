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
	r, t, err := opengl.NewZeroRenderTarget(screenWidth*screenScale, screenHeight*screenScale, true)
	if err != nil {
		return nil, err
	}

	screen, err := idsInstance.createRenderTarget(screenWidth, screenHeight, gl.NEAREST)
	if err != nil {
		return nil, err
	}

	c := &graphicsContext{
		currents:     make([]*RenderTarget, 1),
		defaultR:     &RenderTarget{r, &Texture{t}},
		screen:       screen,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	idsInstance.fillRenderTarget(c.screen, 0, 0, 0)

	return c, nil
}

type graphicsContext struct {
	screen       *RenderTarget
	defaultR     *RenderTarget
	currents     []*RenderTarget
	screenWidth  int
	screenHeight int
	screenScale  int
}

var _ GraphicsContext = new(graphicsContext)

func (c *graphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	glRenderTarget := c.screen.glRenderTarget
	texture := c.screen.texture
	glTexture := texture.glTexture

	glRenderTarget.Dispose()
	glTexture.Dispose()
}

func (c *graphicsContext) Clear() error {
	return c.Fill(0, 0, 0)
}

func (c *graphicsContext) Fill(r, g, b uint8) error {
	return idsInstance.fillRenderTarget(c.currents[len(c.currents)-1], r, g, b)
}

func (c *graphicsContext) DrawTexture(texture *Texture, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	current := c.currents[len(c.currents)-1]
	return idsInstance.drawTexture(current, texture, parts, geo, color)
}

func (c *graphicsContext) PushRenderTarget(renderTarget *RenderTarget) {
	c.currents = append(c.currents, renderTarget)
}

func (c *graphicsContext) PopRenderTarget() {
	c.currents = c.currents[:len(c.currents)-1]
}

func (c *graphicsContext) preUpdate() {
	c.currents = c.currents[0:1]
	c.currents[0] = c.defaultR
	c.PushRenderTarget(c.screen)
	c.Clear()
}

func (c *graphicsContext) postUpdate() {
	c.PopRenderTarget()
	c.Clear()

	scale := float64(c.screenScale)
	geo := GeometryMatrixI()
	geo.Concat(ScaleGeometry(scale, scale))
	DrawWholeTexture(c, c.screen.texture, geo, ColorMatrixI())

	gl.Flush()
}

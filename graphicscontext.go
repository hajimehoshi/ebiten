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
	// The defualt framebuffer should be 0.
	r := opengl.NewRenderTarget(screenWidth*screenScale, screenHeight*screenScale, true)

	screen, err := idsInstance.createRenderTarget(screenWidth, screenHeight, gl.NEAREST)
	if err != nil {
		return nil, err
	}

	c := &graphicsContext{
		currents:     make([]*RenderTarget, 1),
		defaultR:     idsInstance.addRenderTarget(r),
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
	idsInstance.deleteRenderTarget(c.screen)
}

func (c *graphicsContext) Clear() error {
	return c.Fill(0, 0, 0)
}

func (c *graphicsContext) Fill(r, g, b uint8) error {
	return idsInstance.fillRenderTarget(c.currents[len(c.currents)-1], r, g, b)
}

func (c *graphicsContext) Texture(texture *Texture) Drawer {
	return &textureWithContext{texture, c}
}

func (c *graphicsContext) RenderTarget(id *RenderTarget) Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
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
	DrawWhole(c.RenderTarget(c.screen), c.screenWidth, c.screenHeight, geo, ColorMatrixI())

	gl.Flush()
}

type textureWithContext struct {
	texture *Texture
	context *graphicsContext
}

func (t *textureWithContext) Draw(parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	current := t.context.currents[len(t.context.currents)-1]
	return idsInstance.drawTexture(current, t.texture, parts, geo, color)
}

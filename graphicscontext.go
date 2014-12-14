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
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)

	// The defualt framebuffer should be 0.
	r := opengl.NewRenderTarget(screenWidth*screenScale, screenHeight*screenScale, true)

	screenID, err := idsInstance.createRenderTarget(screenWidth, screenHeight, gl.NEAREST)
	if err != nil {
		return nil, err
	}

	c := &graphicsContext{
		currentIDs:   make([]RenderTargetID, 1),
		defaultID:    idsInstance.addRenderTarget(r),
		screenID:     screenID,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	idsInstance.fillRenderTarget(c.screenID, 0, 0, 0)

	return c, nil
}

type graphicsContext struct {
	screenID     RenderTargetID
	defaultID    RenderTargetID
	currentIDs   []RenderTargetID
	screenWidth  int
	screenHeight int
	screenScale  int
}

var _ GraphicsContext = new(graphicsContext)

func (c *graphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	idsInstance.deleteRenderTarget(c.screenID)
}

func (c *graphicsContext) Clear() error {
	return c.Fill(0, 0, 0)
}

func (c *graphicsContext) Fill(r, g, b uint8) error {
	return idsInstance.fillRenderTarget(c.currentIDs[len(c.currentIDs)-1], r, g, b)
}

func (c *graphicsContext) Texture(id TextureID) Drawer {
	return &textureWithContext{id, c}
}

func (c *graphicsContext) RenderTarget(id RenderTargetID) Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *graphicsContext) PushRenderTarget(renderTargetID RenderTargetID) {
	c.currentIDs = append(c.currentIDs, renderTargetID)
}

func (c *graphicsContext) PopRenderTarget() {
	c.currentIDs = c.currentIDs[:len(c.currentIDs)-1]
}

func (c *graphicsContext) preUpdate() {
	c.currentIDs = c.currentIDs[0:1]
	c.currentIDs[0] = c.defaultID
	c.PushRenderTarget(c.screenID)
	c.Clear()
}

func (c *graphicsContext) postUpdate() {
	c.PopRenderTarget()
	c.Clear()

	scale := float64(c.screenScale)
	geo := GeometryMatrixI()
	geo.Scale(scale, scale)
	DrawWhole(c.RenderTarget(c.screenID), c.screenWidth, c.screenHeight, geo, ColorMatrixI())

	gl.Flush()
}

type textureWithContext struct {
	id      TextureID
	context *graphicsContext
}

func (t *textureWithContext) Draw(parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	currentID := t.context.currentIDs[len(t.context.currentIDs)-1]
	return idsInstance.drawTexture(currentID, t.id, parts, geo, color)
}

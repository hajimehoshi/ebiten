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

package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten"
)

func Initialize(screenWidth, screenHeight, screenScale int) (*GraphicsContext, error) {
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)

	c := &GraphicsContext{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	// The defualt framebuffer should be 0.
	c.defaultID = idsInstance.addRenderTarget(&renderTarget{
		width:  screenWidth * screenScale,
		height: screenHeight * screenScale,
		flipY:  true,
	})

	var err error
	c.screenID, err = idsInstance.createRenderTarget(screenWidth, screenHeight, gl.NEAREST)
	if err != nil {
		return nil, err
	}

	// TODO: This is a special stack only for clearing. Can we change this?
	c.currentIDs = []ebiten.RenderTargetID{c.screenID}
	c.Clear()

	return c, nil
}

type GraphicsContext struct {
	screenID     ebiten.RenderTargetID
	defaultID    ebiten.RenderTargetID
	currentIDs   []ebiten.RenderTargetID
	screenWidth  int
	screenHeight int
	screenScale  int
}

var _ ebiten.GraphicsContext = new(GraphicsContext)

func (c *GraphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	idsInstance.deleteRenderTarget(c.screenID)
}

func (c *GraphicsContext) Clear() {
	c.Fill(0, 0, 0)
}

func (c *GraphicsContext) Fill(r, g, b uint8) {
	idsInstance.fillRenderTarget(c.currentIDs[len(c.currentIDs)-1], r, g, b)
}

func (c *GraphicsContext) Texture(id ebiten.TextureID) ebiten.Drawer {
	return &textureWithContext{id, c}
}

func (c *GraphicsContext) RenderTarget(id ebiten.RenderTargetID) ebiten.Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *GraphicsContext) PushRenderTarget(renderTargetID ebiten.RenderTargetID) {
	c.currentIDs = append(c.currentIDs, renderTargetID)
}

func (c *GraphicsContext) PopRenderTarget() {
	c.currentIDs = c.currentIDs[:len(c.currentIDs)-1]
}

func (c *GraphicsContext) PreUpdate() {
	c.currentIDs = []ebiten.RenderTargetID{c.defaultID}
	c.PushRenderTarget(c.screenID)
	c.Clear()
}

func (c *GraphicsContext) PostUpdate() {
	c.PopRenderTarget()
	c.Clear()

	scale := float64(c.screenScale)
	geo := ebiten.GeometryMatrixI()
	geo.Scale(scale, scale)
	ebiten.DrawWhole(c.RenderTarget(c.screenID), c.screenWidth, c.screenHeight, geo, ebiten.ColorMatrixI())

	gl.Flush()
}

type textureWithContext struct {
	id      ebiten.TextureID
	context *GraphicsContext
}

func (t *textureWithContext) Draw(parts []ebiten.TexturePart, geo ebiten.GeometryMatrix, color ebiten.ColorMatrix) {
	currentID := t.context.currentIDs[len(t.context.currentIDs)-1]
	idsInstance.drawTexture(currentID, t.id, parts, geo, color)
}

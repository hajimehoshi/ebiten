package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
)

func Initialize(screenWidth, screenHeight, screenScale int) (*Context, error) {
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)

	c := &Context{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	// The defualt framebuffer should be 0.
	c.defaultId = idsInstance.addRenderTarget(&renderTarget{
		width:  screenWidth * screenScale,
		height: screenHeight * screenScale,
		flipY:  true,
	})

	var err error
	c.screenId, err = idsInstance.createRenderTarget(screenWidth, screenHeight, graphics.FilterNearest)
	if err != nil {
		return nil, err
	}
	c.ResetOffscreen()
	c.Clear()

	return c, nil
}

type Context struct {
	screenId     graphics.RenderTargetID
	defaultId    graphics.RenderTargetID
	currentId    graphics.RenderTargetID
	screenWidth  int
	screenHeight int
	screenScale  int
}

func (c *Context) dispose() {
	// NOTE: Now this method is not used anywhere.
	idsInstance.deleteRenderTarget(c.screenId)
}

func (c *Context) Clear() {
	c.Fill(0, 0, 0)
}

func (c *Context) Fill(r, g, b uint8) {
	idsInstance.fillRenderTarget(c.currentId, r, g, b)
}

func (c *Context) Texture(id graphics.TextureID) graphics.Drawer {
	return &textureWithContext{id, c}
}

func (c *Context) RenderTarget(id graphics.RenderTargetID) graphics.Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *Context) ResetOffscreen() {
	c.currentId = c.screenId
}

func (c *Context) SetOffscreen(renderTargetId graphics.RenderTargetID) {
	c.currentId = renderTargetId
}

func (c *Context) PreUpdate() {
	c.ResetOffscreen()
	c.Clear()
}

func (c *Context) PostUpdate() {
	c.SetOffscreen(c.defaultId)
	c.Clear()

	scale := float64(c.screenScale)
	geo := matrix.GeometryI()
	geo.Scale(scale, scale)
	graphics.DrawWhole(c.RenderTarget(c.screenId), c.screenWidth, c.screenHeight, geo, matrix.ColorI())

	gl.Flush()
}

type textureWithContext struct {
	id      graphics.TextureID
	context *Context
}

func (t *textureWithContext) Draw(parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	idsInstance.drawTexture(t.context.currentId, t.id, parts, geo, color)
}

package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Context struct {
	screenId     graphics.RenderTargetId
	mainId       graphics.RenderTargetId
	currentId    graphics.RenderTargetId
	ids          *ids
	screenWidth  int
	screenHeight int
	screenScale  int
}

func newContext(ids *ids, screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		ids:          ids,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}
	mainRenderTarget := newRTWithCurrentFramebuffer(
		screenWidth*screenScale,
		screenHeight*screenScale)
	context.mainId = context.ids.addRenderTarget(mainRenderTarget)

	var err error
	context.screenId, err = ids.createRenderTarget(
		screenWidth, screenHeight, graphics.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}
	context.ResetOffscreen()
	context.Clear()

	enableAlphaBlending()

	return context
}

func (c *Context) Dispose() {
	// TODO: remove main framebuffer?
	c.ids.deleteRenderTarget(c.screenId)
}

func (c *Context) Update(draw func(graphics.Context)) {
	c.ResetOffscreen()
	c.Clear()

	draw(c)

	c.SetOffscreen(c.mainId)
	c.Clear()

	scale := float64(c.screenScale)
	geo := matrix.IdentityGeometry()
	geo.Scale(scale, scale)
	graphics.DrawWhole(
		c.RenderTarget(c.screenId),
		c.screenWidth,
		c.screenHeight,
		geo,
		matrix.IdentityColor())

	flush()
}

func (c *Context) Clear() {
	c.Fill(0, 0, 0)
}

func (c *Context) Fill(r, g, b uint8) {
	c.ids.fillRenderTarget(c.currentId, r, g, b)
}

func (c *Context) Texture(id graphics.TextureId) graphics.Drawer {
	return &TextureWithContext{id, c}
}

func (c *Context) RenderTarget(id graphics.RenderTargetId) graphics.Drawer {
	return &TextureWithContext{c.ids.toTexture(id), c}
}

func (c *Context) ResetOffscreen() {
	c.currentId = c.screenId
}

func (c *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	c.currentId = renderTargetId
}

type TextureWithContext struct {
	id      graphics.TextureId
	context *Context
}

func (t *TextureWithContext) Draw(
	parts []graphics.TexturePart,
	geo matrix.Geometry,
	color matrix.Color) {
	t.context.ids.drawTexture(t.context.currentId, t.id, parts, geo, color)
}

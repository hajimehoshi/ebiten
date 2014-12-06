package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
	"github.com/hajimehoshi/ebiten/ui"
	"sync"
)

func flush() {
	gl.Flush()
}

var onceInit sync.Once

type Context struct {
	screenId     graphics.RenderTargetId
	defaultId    graphics.RenderTargetId
	currentId    graphics.RenderTargetId
	screenWidth  int
	screenHeight int
	screenScale  int
}

func NewContext(screenWidth, screenHeight, screenScale int) *Context {
	onceInit.Do(func() {
		gl.Init()
		gl.Enable(gl.TEXTURE_2D)
		gl.Enable(gl.BLEND)
	})

	context := &Context{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	defaultRenderTarget := &RenderTarget{
		width:  screenWidth * screenScale,
		height: screenHeight * screenScale,
		flipY:  true,
	}
	context.defaultId = idsInstance.addRenderTarget(defaultRenderTarget)

	var err error
	context.screenId, err = idsInstance.createRenderTarget(screenWidth, screenHeight, graphics.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}
	context.ResetOffscreen()
	context.Clear()

	return context
}

func (c *Context) Dispose() {
	// TODO: remove the default framebuffer?
	idsInstance.deleteRenderTarget(c.screenId)
}

// TODO: This interface is confusing: Can we change this?
func (c *Context) Update(drawer ui.Drawer) error {
	c.ResetOffscreen()
	c.Clear()

	if err := drawer.Draw(c); err != nil {
		return err
	}

	c.SetOffscreen(c.defaultId)
	c.Clear()

	scale := float64(c.screenScale)
	geo := matrix.GeometryI()
	geo.Scale(scale, scale)
	graphics.DrawWhole(c.RenderTarget(c.screenId), c.screenWidth, c.screenHeight, geo, matrix.ColorI())

	flush()
	return nil
}

func (c *Context) Clear() {
	c.Fill(0, 0, 0)
}

func (c *Context) Fill(r, g, b uint8) {
	idsInstance.fillRenderTarget(c.currentId, r, g, b)
}

func (c *Context) Texture(id graphics.TextureId) graphics.Drawer {
	return &textureWithContext{id, c}
}

func (c *Context) RenderTarget(id graphics.RenderTargetId) graphics.Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *Context) ResetOffscreen() {
	c.currentId = c.screenId
}

func (c *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	c.currentId = renderTargetId
}

type textureWithContext struct {
	id      graphics.TextureId
	context *Context
}

func (t *textureWithContext) Draw(parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	idsInstance.drawTexture(t.context.currentId, t.id, parts, geo, color)
}

package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
	"github.com/hajimehoshi/ebiten/ui"
	"sync"
)

type ContextUpdater struct {
	context *context
}

func NewContextUpdater(screenWidth, screenHeight, screenScale int) *ContextUpdater {
	return &ContextUpdater{
		context: newContext(screenWidth, screenHeight, screenScale),
	}
}

func (u *ContextUpdater) Update(drawer ui.Drawer) error {
	return u.context.update(drawer)
}

var onceInit sync.Once

type context struct {
	screenId     graphics.RenderTargetID
	defaultId    graphics.RenderTargetID
	currentId    graphics.RenderTargetID
	screenWidth  int
	screenHeight int
	screenScale  int
}

func newContext(screenWidth, screenHeight, screenScale int) *context {
	onceInit.Do(func() {
		gl.Init()
		gl.Enable(gl.TEXTURE_2D)
		gl.Enable(gl.BLEND)
	})

	c := &context{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	// The defualt framebuffer should be 0.
	defaultRenderTarget := &renderTarget{
		width:  screenWidth * screenScale,
		height: screenHeight * screenScale,
		flipY:  true,
	}
	c.defaultId = idsInstance.addRenderTarget(defaultRenderTarget)

	var err error
	c.screenId, err = idsInstance.newRenderTarget(screenWidth, screenHeight, graphics.FilterNearest)
	if err != nil {
		panic("opengl.newContext: initializing the offscreen failed: " + err.Error())
	}
	c.ResetOffscreen()
	c.Clear()

	return c
}

func (c *context) dispose() {
	// NOTE: Now this method is not used anywhere.
	idsInstance.deleteRenderTarget(c.screenId)
}

func (c *context) Clear() {
	c.Fill(0, 0, 0)
}

func (c *context) Fill(r, g, b uint8) {
	idsInstance.fillRenderTarget(c.currentId, r, g, b)
}

func (c *context) Texture(id graphics.TextureID) graphics.Drawer {
	return &textureWithContext{id, c}
}

func (c *context) RenderTarget(id graphics.RenderTargetID) graphics.Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *context) ResetOffscreen() {
	c.currentId = c.screenId
}

func (c *context) SetOffscreen(renderTargetId graphics.RenderTargetID) {
	c.currentId = renderTargetId
}

func (c *context) update(drawer ui.Drawer) error {
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

	gl.Flush()
	return nil
}

type textureWithContext struct {
	id      graphics.TextureID
	context *context
}

func (t *textureWithContext) Draw(parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	idsInstance.drawTexture(t.context.currentId, t.id, parts, geo, color)
}

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
	c.defaultId = idsInstance.addRenderTarget(&renderTarget{
		width:  screenWidth * screenScale,
		height: screenHeight * screenScale,
		flipY:  true,
	})

	var err error
	c.screenId, err = idsInstance.createRenderTarget(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		return nil, err
	}
	c.ResetOffscreen()
	c.Clear()

	return c, nil
}

type GraphicsContext struct {
	screenId     ebiten.RenderTargetID
	defaultId    ebiten.RenderTargetID
	currentId    ebiten.RenderTargetID
	screenWidth  int
	screenHeight int
	screenScale  int
}

var _ ebiten.GraphicsContext = new(GraphicsContext)

func (c *GraphicsContext) dispose() {
	// NOTE: Now this method is not used anywhere.
	idsInstance.deleteRenderTarget(c.screenId)
}

func (c *GraphicsContext) Clear() {
	c.Fill(0, 0, 0)
}

func (c *GraphicsContext) Fill(r, g, b uint8) {
	idsInstance.fillRenderTarget(c.currentId, r, g, b)
}

func (c *GraphicsContext) Texture(id ebiten.TextureID) ebiten.Drawer {
	return &textureWithContext{id, c}
}

func (c *GraphicsContext) RenderTarget(id ebiten.RenderTargetID) ebiten.Drawer {
	return &textureWithContext{idsInstance.toTexture(id), c}
}

func (c *GraphicsContext) ResetOffscreen() {
	c.currentId = c.screenId
}

func (c *GraphicsContext) SetOffscreen(renderTargetId ebiten.RenderTargetID) {
	c.currentId = renderTargetId
}

func (c *GraphicsContext) PreUpdate() {
	c.ResetOffscreen()
	c.Clear()
}

func (c *GraphicsContext) PostUpdate() {
	c.SetOffscreen(c.defaultId)
	c.Clear()

	scale := float64(c.screenScale)
	geo := ebiten.GeometryMatrixI()
	geo.Scale(scale, scale)
	ebiten.DrawWhole(c.RenderTarget(c.screenId), c.screenWidth, c.screenHeight, geo, ebiten.ColorMatrixI())

	gl.Flush()
}

type textureWithContext struct {
	id      ebiten.TextureID
	context *GraphicsContext
}

func (t *textureWithContext) Draw(parts []ebiten.TexturePart, geo ebiten.GeometryMatrix, color ebiten.ColorMatrix) {
	idsInstance.drawTexture(t.context.currentId, t.id, parts, geo, color)
}

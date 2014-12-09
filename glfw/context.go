package glfw

import (
	"github.com/hajimehoshi/ebiten"
)

type context struct {
	canvas *canvas
}

var _ ebiten.GraphicsContext = new(context)

func (c *context) Clear() {
	c.canvas.use(func() {
		c.canvas.context.Clear()
	})
}

func (c *context) Fill(r, g, b uint8) {
	c.canvas.use(func() {
		c.canvas.context.Fill(r, g, b)
	})
}

func (c *context) Texture(id ebiten.TextureID) (d ebiten.Drawer) {
	c.canvas.use(func() {
		d = &drawer{
			canvas:      c.canvas,
			innerDrawer: c.canvas.context.Texture(id),
		}
	})
	return
}

func (c *context) RenderTarget(id ebiten.RenderTargetID) (d ebiten.Drawer) {
	c.canvas.use(func() {
		d = &drawer{
			canvas:      c.canvas,
			innerDrawer: c.canvas.context.RenderTarget(id),
		}
	})
	return
}

func (c *context) ResetOffscreen() {
	c.canvas.use(func() {
		c.canvas.context.ResetOffscreen()
	})
}

func (c *context) SetOffscreen(id ebiten.RenderTargetID) {
	c.canvas.use(func() {
		c.canvas.context.SetOffscreen(id)
	})
}

type drawer struct {
	canvas      *canvas
	innerDrawer ebiten.Drawer
}

var _ ebiten.Drawer = new(drawer)

func (d *drawer) Draw(parts []ebiten.TexturePart, geo ebiten.GeometryMatrix, color ebiten.ColorMatrix) {
	d.canvas.use(func() {
		d.innerDrawer.Draw(parts, geo, color)
	})
}

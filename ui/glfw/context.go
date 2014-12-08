package glfw

import (
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
)

type context struct {
	canvas *canvas
}

var _ graphics.Context = new(context)

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

func (c *context) Texture(id graphics.TextureID) (d graphics.Drawer) {
	c.canvas.use(func() {
		d = &drawer{
			canvas:      c.canvas,
			innerDrawer: c.canvas.context.Texture(id),
		}
	})
	return
}

func (c *context) RenderTarget(id graphics.RenderTargetID) (d graphics.Drawer) {
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

func (c *context) SetOffscreen(id graphics.RenderTargetID) {
	c.canvas.use(func() {
		c.canvas.context.SetOffscreen(id)
	})
}

type drawer struct {
	canvas      *canvas
	innerDrawer graphics.Drawer
}

var _ graphics.Drawer = new(drawer)

func (d *drawer) Draw(parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	d.canvas.use(func() {
		d.innerDrawer.Draw(parts, geo, color)
	})
}

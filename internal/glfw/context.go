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

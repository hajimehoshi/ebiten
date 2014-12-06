package graphics

import (
	"github.com/hajimehoshi/ebiten/graphics/matrix"
)

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type TexturePart struct {
	LocationX int
	LocationY int
	Source    Rect
}

type Drawer interface {
	Draw(parts []TexturePart, geometryMatrix matrix.Geometry, colorMatrix matrix.Color)
}

func DrawWhole(drawer Drawer, width, height int, geo matrix.Geometry, color matrix.Color) {
	parts := []TexturePart{
		{0, 0, Rect{0, 0, width, height}},
	}
	drawer.Draw(parts, geo, color)
}

type Context interface {
	Clear()
	Fill(r, g, b uint8)
	Texture(id TextureID) Drawer
	RenderTarget(id RenderTargetID) Drawer

	ResetOffscreen()
	SetOffscreen(id RenderTargetID)
}

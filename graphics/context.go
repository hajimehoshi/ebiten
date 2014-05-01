package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Drawer interface {
	Draw(geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawParts(parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
}

type Context interface {
	Clear()
	Fill(r, g, b uint8)
	Texture(id TextureId) Drawer
	RenderTarget(id RenderTargetId) Drawer

	ResetOffscreen()
	SetOffscreen(id RenderTargetId)

	// TODO: glTextureSubImage2D
}

package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Context interface {
	Clear()
	Fill(r, g, b uint8)
	// TODO: Refacotring
	DrawTexture(id TextureId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawRenderTarget(id RenderTargetId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(id TextureId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawRenderTargetParts(id RenderTargetId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)

	ResetOffscreen()
	SetOffscreen(id RenderTargetId)

	// TODO: glTextureSubImage2D
}

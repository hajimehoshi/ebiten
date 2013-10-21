package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
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

type Context interface {
	ToTexture(id RenderTargetId) TextureId

	Clear()
	Fill(r, g, b uint8)
	DrawTexture(id TextureId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(id TextureId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	ResetOffscreen()
	SetOffscreen(id RenderTargetId)
}

type TextureFactory interface {
	NewRenderTarget(width, height int) (RenderTargetId, error)
	NewTextureFromImage(img image.Image) (TextureId, error)
}

type TextureId int

// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetId int

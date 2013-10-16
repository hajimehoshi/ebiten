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
	TextureID(renderTargetID RenderTargetID) TextureID

	Clear()
	Fill(r, g, b uint8)
	DrawTexture(textureID TextureID,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(textureID TextureID,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	ResetOffscreen()
	SetOffscreen(renderTargetID RenderTargetID)
}

type TextureFactory interface {
	NewRenderTarget(width, height int) RenderTargetID
	NewTextureFromImage(img image.Image) (TextureID, error)
}

type TextureID int

// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetID int

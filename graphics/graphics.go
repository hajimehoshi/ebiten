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
	NewRenderTarget(width, height int) RenderTarget
	NewTextureFromImage(img image.Image) (Texture, error)
}

type Texture interface {
	ID() TextureID
}

type TextureID int

// The interface of a render target. This is essentially same as a texture, but
// it is assumed that the all alpha of a render target is maximum.
type RenderTarget interface {
	Texture() Texture
	ID() RenderTargetID
}

type RenderTargetID int

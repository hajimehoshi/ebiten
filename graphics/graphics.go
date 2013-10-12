package graphics

import (
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
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
	Screen() RenderTarget
	Clear()
	Fill(r, g, b uint8)
	DrawTexture(textureID TextureID,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(textureID TextureID,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	SetOffscreen(renderTargetID RenderTargetID)
}

type TextureFactory interface {
	NewRenderTarget(width, height int) RenderTarget
	NewTextureFromImage(img image.Image) (Texture, error)
}

type Texture interface {
	ID() TextureID
	Width() int
	Height() int
}

type TextureID int

type RenderTarget interface {
	Texture() Texture
	ID() RenderTargetID
	Width() int
	Height() int
}

type RenderTargetID int

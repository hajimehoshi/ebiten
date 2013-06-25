package graphics

import (
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
)

type Device interface {
	Update()
	TextureFactory() TextureFactory
	OffscreenTexture() Texture
}

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

type GraphicsContext interface {
	Clear()
	Fill(clr color.Color)
	DrawRect(rect Rect, clr color.Color)
	DrawTexture(textureID TextureID,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(textureID TextureID,
		locations []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	SetOffscreen(textureID TextureID)
}

type TextureFactory interface {
	NewTexture(width, height int) Texture
	NewTextureFromImage(img image.Image) (Texture, error)
}

type Texture struct {
	ID     TextureID
	Width  int
	Height int
}

type TextureID int

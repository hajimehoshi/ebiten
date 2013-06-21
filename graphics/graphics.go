package graphics

import (
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
)

type Device interface {
	Update()
	TextureFactory() TextureFactory
}

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type TextureLocation struct {
	LocationX int
	LocationY int
	Source    Rect
}

type GraphicsContext interface {
	Clear()
	Fill(color color.Color)
	DrawTexture(texture Texture,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(textureId TextureID,
		locations []TextureLocation,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	SetOffscreen(textureId TextureID)
}

type TextureFactory interface {
	NewTexture(width, height int) Texture
	NewTextureFromImage(img image.Image) Texture
}

type Texture struct {
	ID     TextureID
	Width  int
	Height int
}

type TextureID int

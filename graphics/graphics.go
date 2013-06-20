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

type GraphicsContext interface {
	Clear()
	Fill(color color.Color)
	DrawTexture(textureId TextureID,
		srcX, srcY, srcWidth, srcHeight int,
		geometryMatrix matrix.Geometry, colorMatrix matrix.Color)
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

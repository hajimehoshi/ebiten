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

type Point struct {
	X int
	Y int
}

type Size struct {
	Width int
	Height int
}

type Rectangle struct {
	Location Point
	Size Size
}

type TextureLocation struct {
	Location Point
	Source Rectangle
}

type GraphicsContext interface {
	Clear()
	Fill(color color.Color)
	DrawTexture(textureId TextureID,
		source Rectangle,
		geometryMatrix matrix.Geometry, colorMatrix matrix.Color)
	DrawTextures(textureId TextureID,
		locations []TextureLocation,
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

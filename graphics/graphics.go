package graphics

import (
	"image"
	"image/color"
)

type Device interface {
	Update()
}

type GraphicsContext interface {
	Clear()
	Fill(color color.Color)
	DrawTexture(texture *Texture,
		srcX, srcY, srcWidth, srcHeight int,
		geometryMatrix *GeometryMatrix, colorMatrix *ColorMatrix)
	SetOffscreen(texture *Texture)
}

type Texture struct {
	Width int
	Height int
	Image image.Image
}

func NewTexture(width, height int) *Texture {
	return &Texture{width, height, nil}
}

func NewTextureFromImage(img image.Image) *Texture {
	size := img.Bounds().Size()
	return &Texture{size.X, size.Y, img}
}

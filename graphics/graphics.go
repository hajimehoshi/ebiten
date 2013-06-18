package graphics

import (
	"image"
	"image/color"
)

type Device interface {
	Update()
	// TODO: Move somewhere
	NewTexture(width, height int) Texture
	NewTextureFromImage(img image.Image) Texture
}

type GraphicsContext interface {
	Clear()
	Fill(color color.Color)
	DrawTexture(texture Texture,
		srcX, srcY, srcWidth, srcHeight int,
		geometryMatrix *GeometryMatrix, colorMatrix *ColorMatrix)
	SetOffscreen(texture Texture)
}

type Texture interface {
	Width() int
	Height() int
	TextureWidth() int
	TextureHeight() int
}

package graphics

import (
	"image"
)

type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

type TextureID int

// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetID int

var currentTextureFactory TextureFactory

type TextureFactory interface {
	NewRenderTargetID(width, height int, filter Filter) (RenderTargetID, error)
	NewTextureID(img image.Image, filter Filter) (TextureID, error)
}

func SetTextureFactory(textureFactory TextureFactory) {
	currentTextureFactory = textureFactory
}

func NewRenderTargetID(width, height int, filter Filter) (RenderTargetID, error) {
	if currentTextureFactory == nil {
		panic("graphics.NewRenderTarget: currentTextureFactory is not set.")
	}
	return currentTextureFactory.NewRenderTargetID(width, height, filter)
}

func NewTextureID(img image.Image, filter Filter) (TextureID, error) {
	if currentTextureFactory == nil {
		panic("graphics.NewTexture: currentTextureFactory is not set")
	}
	return currentTextureFactory.NewTextureID(img, filter)
}

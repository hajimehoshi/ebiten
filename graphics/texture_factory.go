package graphics

import (
	"image"
)

type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

type TextureId int

// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetId int

type TextureFactory interface {
	CreateRenderTarget(width, height int, filter Filter) (RenderTargetId, error)
	CreateTexture(img image.Image, filter Filter) (TextureId, error)
}

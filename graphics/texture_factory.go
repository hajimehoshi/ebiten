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

type TextureCreatedEvent struct {
	Id    TextureId
	Error error
}

type RenderTargetCreatedEvent struct {
	Id    RenderTargetId
	Error error
}

type TextureFactory interface {
	CreateRenderTarget(width, height int, filter Filter) <-chan RenderTargetCreatedEvent
	CreateTexture(img image.Image, filter Filter) <-chan TextureCreatedEvent
}

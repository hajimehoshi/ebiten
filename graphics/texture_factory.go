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
	Tag   interface{}
	Id    TextureId
	Error error
}

type RenderTargetCreatedEvent struct {
	Tag   interface{}
	Id    RenderTargetId
	Error error
}

type TextureFactory interface {
	CreateRenderTarget(tag interface{}, width, height int, filter Filter)
	CreateTexture(tag interface{}, img image.Image, filter Filter)
	Events() <-chan interface{}
}

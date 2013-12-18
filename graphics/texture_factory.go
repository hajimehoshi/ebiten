package graphics

import (
	"image"
)

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
	CreateRenderTarget(tag interface{}, width, height int)
	CreateTexture(tag interface{}, img image.Image, filter Filter)
	Events() <-chan interface{}
}

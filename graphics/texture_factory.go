package graphics

import (
	"image"
)

type TextureCreatedEvent struct {
	Tag   string
	Id    TextureId
	Error error
}

type RenderTargetCreatedEvent struct {
	Tag            string
	RenderTargetId RenderTargetId
	Error          error
}

type TextureFactoryEvents interface {
	TextureCreated() <-chan TextureCreatedEvent
	RenderTargetCreated() <-chan RenderTargetCreatedEvent
}

type TextureFactory interface {
	CreateRenderTarget(tag string, width, height int) (RenderTargetId, error)
	CreateTextureFromImage(tag string, img image.Image) (TextureId, error)
	//TextureFactoryEvents
}

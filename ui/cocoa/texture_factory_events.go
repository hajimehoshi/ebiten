package cocoa

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type textureFactoryEvents struct {
	textureCreated      chan graphics.TextureCreatedEvent
	renderTargetCreated chan graphics.RenderTargetCreatedEvent
}

func (t *textureFactoryEvents) TextureCreated() <-chan graphics.TextureCreatedEvent {
	if t.textureCreated != nil {
		return t.textureCreated
	}
	t.textureCreated = make(chan graphics.TextureCreatedEvent)
	return t.textureCreated
}

func (t *textureFactoryEvents) notifyTextureCreated(e graphics.TextureCreatedEvent) {
	if t.textureCreated == nil {
		return
	}
	go func() {
		t.textureCreated <- e
	}()
}

func (t *textureFactoryEvents) RenderTargetCreated() <-chan graphics.RenderTargetCreatedEvent {
	if t.renderTargetCreated != nil {
		return t.renderTargetCreated
	}
	t.renderTargetCreated = make(chan graphics.RenderTargetCreatedEvent)
	return t.renderTargetCreated
}

func (t *textureFactoryEvents) notifyRenderTargetCreated(e graphics.RenderTargetCreatedEvent) {
	if t.renderTargetCreated == nil {
		return
	}
	go func() {
		t.renderTargetCreated <- e
	}()
}

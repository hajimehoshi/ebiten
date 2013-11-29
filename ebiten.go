package ebiten

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type ScreenSizeUpdatedEvent struct {
	Width  int
	Height int
}

type InputStateUpdatedEvent struct {
	X int
	Y int
}

type UI interface {
	PollEvents()
	InitTextures(func(graphics.TextureFactory))
	Draw(func(graphics.Canvas))

	ScreenSizeUpdated() <-chan ScreenSizeUpdatedEvent
	InputStateUpdated() <-chan InputStateUpdatedEvent
}

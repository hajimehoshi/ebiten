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

type UIEvents interface {
	ObserveScreenSizeUpdated() <-chan ScreenSizeUpdatedEvent
	ObserveInputStateUpdated() <-chan InputStateUpdatedEvent
}

type UI interface {
	PollEvents()
	InitTextures(func(graphics.TextureFactory))
	Draw(func(graphics.Canvas))
	UIEvents
}

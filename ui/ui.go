package ui

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
	ScreenSizeUpdated() <-chan ScreenSizeUpdatedEvent
	InputStateUpdated() <-chan InputStateUpdatedEvent
}

type UI interface {
	PollEvents()
	LoadResources(func(graphics.TextureFactory))
	LoadTextures(map[int]string)
	Draw(func(graphics.Canvas))
	UIEvents
}

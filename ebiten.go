package ebiten

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

// TODO: Remove this
type GameContext interface {
	ScreenWidth() int
	ScreenHeight() int
}

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
}

type UI interface {
	PollEvents()
	InitTextures(func(graphics.TextureFactory))
	Draw(func(graphics.Canvas))

	InputStateUpdated() <-chan InputStateUpdatedEvent

	// TODO: Remove this
	Update(func(GameContext))
}

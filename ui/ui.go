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
	Draw(func(graphics.Canvas))
	UIEvents
}

type WindowEvents interface {
	ScreenSizeUpdated() <-chan ScreenSizeUpdatedEvent
	InputStateUpdated() <-chan InputStateUpdatedEvent
}

type Window interface {
	Draw(func(graphics.Canvas))
	WindowEvents
}

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

type WindowClosedEvent struct {
}

type UI interface {
	PollEvents()
	CreateWindow(screenWidth, screenHeight, screenScale int, title string) Window
}

type WindowEvents interface {
	ScreenSizeUpdated() <-chan ScreenSizeUpdatedEvent
	InputStateUpdated() <-chan InputStateUpdatedEvent
	WindowClosed() <-chan WindowClosedEvent
}

type Window interface {
	Draw(func(graphics.Canvas))
	WindowEvents
}

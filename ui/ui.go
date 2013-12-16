package ui

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
)

type ScreenSizeUpdatedEvent struct {
	Width  int
	Height int
}

type KeyStateUpdatedEvent struct {
	Keys []Key
}

type MouseStateUpdatedEvent struct {
	X int
	Y int
}

type WindowClosedEvent struct {
}

type UI interface {
	PollEvents()
	CreateWindow(screenWidth, screenHeight, screenScale int, title string) Window
}

type Window interface {
	Draw(func(graphics.Context))
	Events() <-chan interface{}
}

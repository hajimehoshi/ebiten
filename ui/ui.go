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
	KeySpace
	KeyMax
)

type WindowSizeUpdatedEvent struct {
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
	DoEvents()
	CreateGameWindow(screenWidth, screenHeight, screenScale int, title string) GameWindow
	RunMainLoop()
}

type Window interface {
	Events() <-chan interface{}
}

type GameWindow interface {
	Draw(func(graphics.Context))
	Window
}

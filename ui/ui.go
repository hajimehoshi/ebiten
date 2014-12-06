package ui

import (
	"github.com/hajimehoshi/ebiten/graphics"
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

type UI interface {
	Start(widht, height, scale int, title string) (Canvas, graphics.TextureFactory)
	DoEvents()
	Terminate()
}

type Keys interface {
	Includes(key Key) bool
}

type InputState interface {
	PressedKeys() Keys
	MouseX() int
	MouseY() int
}

type Canvas interface {
	Draw(func(graphics.Context))
	IsClosed() bool
	InputState() InputState
}

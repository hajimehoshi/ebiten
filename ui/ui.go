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

type UI interface {
	CreateCanvas(widht, height, scale int, title string) Canvas
	Start()
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

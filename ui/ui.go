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
	Start(widht, height, scale int, title string) Canvas
	DoEvents()
	Terminate()
}

type InputState interface {
	IsPressedKey(key Key) bool
	MouseX() int
	MouseY() int
}

type Canvas interface {
	graphics.TextureFactory
	Draw(func(graphics.Context))
	IsClosed() bool
	InputState() InputState
}

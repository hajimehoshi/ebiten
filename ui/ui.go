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

type CanvasState struct {
	Width    int
	Height   int
	Scale    int
	Keys     []Key
	MouseX   int
	MouseY   int
	IsClosed bool
}

type Canvas interface {
	Draw(func(graphics.Context))
	State() CanvasState
}

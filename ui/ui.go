package ui

import (
	"github.com/hajimehoshi/ebiten/graphics"
)

type UI interface {
	Start(widht, height, scale int, title string) Canvas
	DoEvents()
	Terminate()
}

type Canvas interface {
	graphics.TextureFactory
	Draw(func(graphics.Context))
	IsClosed() bool
}

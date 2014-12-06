package ui

import (
	"github.com/hajimehoshi/ebiten/graphics"
)

type UI interface {
	Start(widht, height, scale int, title string) (Canvas, error)
	DoEvents()
	Terminate()
}

type Drawer interface {
	Draw(c graphics.Context) error
}

type Canvas interface {
	Draw(drawer Drawer) error
	IsClosed() bool
}

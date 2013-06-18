package ui

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type UI interface {
	ScreenWidth() int
	ScreenHeight() int
	ScreenScale() int
	Run(device graphics.Device)
}

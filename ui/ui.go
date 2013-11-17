package ui

import (
	"github.com/hajimehoshi/go-ebiten"
)

type UI interface {
	Run(game ebiten.Game)
}

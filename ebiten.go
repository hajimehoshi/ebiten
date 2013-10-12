package ebiten

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
)

const (
	FPS = 60
)

type Game interface {
	Init(tf graphics.TextureFactory)
	Update(context GameContext)
	Draw(context graphics.Context)
}

type GameContext interface {
	ScreenWidth() int
	ScreenHeight() int
	InputState() InputState
	Terminate()
}

type InputState struct {
	X int
	Y int
}

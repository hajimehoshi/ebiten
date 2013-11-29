package ebiten

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type Game interface {
	InitTextures(tf graphics.TextureFactory)
	Update(context GameContext)
	Draw(canvas graphics.Canvas)
}

type GameContext interface {
	ScreenWidth() int
	ScreenHeight() int
	InputState() InputState
}

type InputState struct {
	X int
	Y int
}

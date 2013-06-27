package ebiten

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
)

type TapInfo struct {
	X int
	Y int
}

type Game interface {
	ScreenWidth() int
	ScreenHeight() int
	Fps() int
	Init(tf graphics.TextureFactory)
	Update(input InputState)
	Draw(g graphics.Context, offscreen graphics.Texture)
}

type UI interface {
	Run()
}

type InputState struct {
	IsTapped bool
	X        int
	Y        int
}

package blank

import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
)

type Blank struct {
}

func New() *Blank {
	return &Blank{}
}

func (game *Blank) ScreenWidth() int {
	return 256
}

func (game *Blank) ScreenHeight() int {
	return 240
}

func (game *Blank) Fps() int {
	return 60
}

func (game *Blank) Init(tf graphics.TextureFactory) {
}

func (game *Blank) Update(input ebiten.InputState) {
}

func (game *Blank) Draw(g graphics.Context, offscreen graphics.Texture) {
}

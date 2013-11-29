package blank

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type Blank struct {
}

func New() *Blank {
	return &Blank{}
}

func (game *Blank) InitTextures(tf graphics.TextureFactory) {
}

func (game *Blank) Update() {
}

func (game *Blank) Draw(canvas graphics.Canvas) {
}

package game

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
)

type Sprites struct {
	ebitenTexture graphics.Texture
}

func NewSprites() *Sprites {
	return &Sprites{}
}

func (game *Sprites) Init(tf graphics.TextureFactory) {
}

func (game *Sprites) Update() {
}

func (game *Sprites) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
}

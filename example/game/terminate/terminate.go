package terminate

import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
)

type Terminate struct {
	life int
}

func New() *Terminate {
	return &Terminate{60}
}

func (game *Terminate) Init(tf graphics.TextureFactory) {
}

func (game *Terminate) Update(context ebiten.GameContext) {
	game.life--
	if game.life <= 0 {
		context.Terminate()
	}
}

func (game *Terminate) Draw(context graphics.Context) {
}

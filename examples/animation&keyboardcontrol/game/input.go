package game

import "github.com/hajimehoshi/ebiten/v2"

const (
	keynone = iota
	keyup
	keydown
	keyleft
	keyright
)

type Input struct {
	msg   string
	state int
}

func NewInput() *Input {
	return &Input{
		msg:   "",
		state: keynone,
	}
}

func (i *Input) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		i.state = keyup
		i.msg = "up pressed"
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		i.state = keydown
		i.msg = "down pressed"
	} else if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		i.state = keyleft
		i.msg = "left pressed"
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		i.state = keyright
		i.msg = "right pressed"
	} else {
		i.state = keynone
		i.msg = "none pressed"
	}
}

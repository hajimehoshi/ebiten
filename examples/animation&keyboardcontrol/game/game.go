package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 1000
	screenHeight = 800
)

type Game struct {
	input     *Input
	man       *Man
	obstacle  *Obstacle
	counter15 int
	counter60 int
}

func (g *Game) Update() error {
	g.input.Update()
	g.man.Update(g.input, g.obstacle)
	if g.input.state != keynone {
		g.counter60++
	}
	if g.counter60 == 4 {
		g.counter60 = 0
		g.counter15++
		if g.counter15 == 4 {
			g.counter15 = 0
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.man.Draw(screen, g.input.state, g.counter15)
	g.obstacle.Draw(screen)
}

func (g *Game) Layout(int, int) (int, int) {
	return screenWidth, screenHeight
}

func NewGame() *Game {
	i := NewInput()
	m, err1 := NewMan()
	if err1 != nil {
		panic(err1)
	}
	o, err2 := NewObstacle()
	if err2 != nil {
		panic(err2)
	}
	return &Game{
		input:    i,
		man:      m,
		obstacle: o,
	}
}

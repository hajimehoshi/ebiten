package blocks

import (
	"github.com/hajimehoshi/ebiten"
)

type Input struct {
	states map[ebiten.Key]int
}

func NewInput() *Input {
	states := map[ebiten.Key]int{}
	for key := ebiten.Key(0); key < ebiten.KeyMax; key++ {
		states[key] = 0
	}
	return &Input{
		states: states,
	}
}

func (i *Input) StateForKey(key ebiten.Key) int {
	return i.states[key]
}

func (i *Input) Update() {
	for key := range i.states {
		if !ebiten.IsKeyPressed(key) {
			i.states[key] = 0
			continue
		}
		i.states[key] += 1
	}
}

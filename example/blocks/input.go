package blocks

import (
	"github.com/hajimehoshi/ebiten/input"
)

type Input struct {
	states map[input.Key]int
}

func NewInput() *Input {
	states := map[input.Key]int{}
	for key := input.Key(0); key < input.KeyMax; key++ {
		states[key] = 0
	}
	return &Input{
		states: states,
	}
}

func (i *Input) StateForKey(key input.Key) int {
	return i.states[key]
}

func (i *Input) Update() {
	for key := range i.states {
		if !input.IsKeyPressed(key) {
			i.states[key] = 0
			continue
		}
		i.states[key] += 1
	}
}

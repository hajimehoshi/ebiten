package blocks

import (
	"github.com/hajimehoshi/ebiten/ui"
)

type Input struct {
	states map[ui.Key]int
}

func NewInput() *Input {
	states := map[ui.Key]int{}
	for key := ui.Key(0); key < ui.KeyMax; key++ {
		states[key] = 0
	}
	return &Input{
		states: states,
	}
}

func (i *Input) StateForKey(key ui.Key) int {
	return i.states[key]
}

func (i *Input) Update(keys ui.Keys) {
	for key := range i.states {
		if !keys.Includes(key) {
			i.states[key] = 0
			continue
		}
		i.states[key] += 1
	}
}

package blocks

import (
	"github.com/hajimehoshi/go-ebiten/ui"
)

type Input struct {
	states          map[ui.Key]int
	lastPressedKeys map[ui.Key]struct{}
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

func (i *Input) Update() {
	for key, _ := range i.states {
		if _, ok := i.lastPressedKeys[key]; !ok {
			i.states[key] = 0
			continue
		}
		i.states[key] += 1
	}
}

func (i *Input) UpdateKeys(keys []ui.Key) {
	i.lastPressedKeys = map[ui.Key]struct{}{}
	for _, key := range keys {
		i.lastPressedKeys[key] = struct{}{}
	}
}

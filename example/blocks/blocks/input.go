// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

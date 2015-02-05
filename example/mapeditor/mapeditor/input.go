// Copyright 2015 Hajime Hoshi
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

package mapeditor

import (
	"github.com/hajimehoshi/ebiten"
)

type Input struct {
	mouseButtonStates [4]int
}

func (i *Input) Update() {
	for b, _ := range i.mouseButtonStates {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButton(b)) {
			i.mouseButtonStates[b] = 0
			continue
		}
		i.mouseButtonStates[b]++
	}
}

func (i *Input) MouseButtonState(m ebiten.MouseButton) int {
	return i.mouseButtonStates[m]
}

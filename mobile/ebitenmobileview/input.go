// Copyright 2016 Hajime Hoshi
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

//go:build android || ios

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type position struct {
	x int
	y int
}

var (
	keys    = map[ui.Key]struct{}{}
	runes   []rune
	touches = map[ui.TouchID]position{}
)

var (
	touchSlice []ui.TouchForInput
)

func updateInput() {
	touchSlice = touchSlice[:0]
	for id, position := range touches {
		touchSlice = append(touchSlice, ui.TouchForInput{
			ID: id,
			X:  float64(position.x),
			Y:  float64(position.y),
		})
	}

	ui.Get().UpdateInput(keys, runes, touchSlice)
}

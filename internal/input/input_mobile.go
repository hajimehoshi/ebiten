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

// +build android ios

package input

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/driver"
)

type Input struct {
	cursorX  int
	cursorY  int
	gamepads [16]gamePad
	touches  map[int]pos
	m        sync.RWMutex
}

func (i *Input) RuneBuffer() []rune {
	return nil
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	return false
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}

func (i *Input) IsMouseButtonPressed(key driver.MouseButton) bool {
	return false
}

func (i *Input) Update(touches []*driver.Touch) {
	i.m.Lock()
	i.touches = map[int]pos{}
	for _, t := range touches {
		i.touches[t.ID] = pos{
			X: t.X,
			Y: t.Y,
		}
	}
	i.m.Unlock()
}

func (i *Input) ResetForFrame() {
	// Do nothing
}

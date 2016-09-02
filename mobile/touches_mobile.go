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

// +build android ios darwin,arm darwin,arm64

package mobile

import (
	"github.com/hajimehoshi/ebiten/internal/ui"
)

type position struct {
	x int
	y int
}

var (
	touches = map[int]position{}
)

// touch implements ui.Touch.
type touch struct {
	id       int
	position position
}

func (t touch) ID() int {
	return t.id
}

func (t touch) Position() (int, int) {
	// TODO: Is this OK to adjust the position here?
	return int(float64(t.position.x) / ui.ScreenScale()),
		int(float64(t.position.y) / ui.ScreenScale())
}

func updateTouches() {
	ts := []ui.Touch{}
	for id, position := range touches {
		ts = append(ts, touch{id, position})
	}
	ui.UpdateTouches(ts)
}

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

package ui

import (
	"sync"
)

type input struct {
	cursorX  int
	cursorY  int
	gamepads [16]gamePad
	touches  []touch
	m        sync.RWMutex
}

func (i *input) IsKeyPressed(key Key) bool {
	return false
}

func (i *input) IsMouseButtonPressed(key MouseButton) bool {
	return false
}

func (i *input) updateTouches(touches []Touch) {
	i.m.Lock()
	defer i.m.Unlock()
	ts := make([]touch, len(touches))
	for i := 0; i < len(ts); i++ {
		ts[i].id = touches[i].ID()
		x, y := touches[i].Position()
		ts[i].x, ts[i].y = x, y
	}
	i.touches = ts
}

// Copyright 2024 The Ebitengine Authors
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

//go:build (!darwin && !js && !windows) || ios

package textinput

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type textInput struct {
	rs       []rune
	lastTick int64
}

var theTextInput textInput

func (t *textInput) Start(x, y int) (<-chan State, func()) {
	// AppendInputChars is updated only when the tick is updated.
	// If the tick is not updated, return nil immediately.
	tick := ebiten.Tick()
	if t.lastTick == tick {
		return nil, nil
	}
	defer func() {
		t.lastTick = tick
	}()

	s := newSession()

	// This is a pseudo implementation with AppendInputChars without IME.
	// This is tentative and should be replaced with IME in the future.
	t.rs = ebiten.AppendInputChars(t.rs[:0])
	if len(t.rs) == 0 {
		return nil, nil
	}
	s.ch <- State{
		Text:      string(t.rs),
		Committed: true,
	}
	// Keep the channel as end() resets s.ch.
	ch := s.ch
	s.end()
	return ch, nil
}

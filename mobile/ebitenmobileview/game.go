// Copyright 2019 The Ebiten Authors
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

package ebitenmobileview

import (
	"sync"

	"github.com/hajimehoshi/ebiten"
)

var theState state

// game is not exported since gomobile complains.
// TODO: Report this error.
type game interface {
	Update(*ebiten.Image) error
	Layout(viewWidth, viewHeight int) (screenWidth, screenHeight int)
}

type state struct {
	game game

	delayedLayout func()

	errorCh <-chan error

	// m is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	m sync.Mutex
}

func SetGame(game game) {
	theState.m.Lock()
	defer theState.m.Unlock()

	theState.game = game

	if theState.delayedLayout != nil {
		theState.delayedLayout()
		theState.delayedLayout = nil
	}
}

func (s *state) isRunning() bool {
	return s.errorCh != nil
}

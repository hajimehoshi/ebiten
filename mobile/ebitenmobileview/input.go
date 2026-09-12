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
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type position struct {
	x float64
	y float64
}

var (
	// inputMu protects the variables below and ptrToID in input_ios.go.
	// The platform entry points are expected to run on the UI thread only;
	// inputMu makes that confinement explicit.
	inputMu sync.Mutex

	keyPressedTimes  [ui.KeyMax + 1]ui.InputTime
	keyReleasedTimes [ui.KeyMax + 1]ui.InputTime
	touches          = map[ui.TouchID]position{}

	// capsLock and numLock stay unknown until a physical keyboard reports them.
	capsLock ui.LockKeyState
	numLock  ui.LockKeyState

	touchSlice []ui.TouchForInput
)

// setKeyReleased records a key release. The release of a key that is not down
// is ignored: the game never saw the key pressed.
//
// setKeyReleased must be called with inputMu held.
func setKeyReleased(key ui.Key) {
	if keyPressedTimes[key] <= keyReleasedTimes[key] {
		return
	}
	keyReleasedTimes[key] = ui.Get().InputTime()
}

// updateInput copies the guarded state to the platform input state.
//
// updateInput must be called with inputMu held.
func updateInput(runes []rune) {
	touchSlice = touchSlice[:0]
	for id, position := range touches {
		touchSlice = append(touchSlice, ui.TouchForInput{
			ID: id,
			X:  position.x,
			Y:  position.y,
		})
	}

	ui.Get().UpdateInput(keyPressedTimes, keyReleasedTimes, runes, touchSlice, capsLock, numLock)
}

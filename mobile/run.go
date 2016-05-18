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

package mobile

import (
	"runtime"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

var chError <-chan error

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height, scale int, title string) {
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, title)
	return
}

func LastErrorString() string {
	select {
	case err := <-chError:
		return err.Error()
	default:
		return ""
	}
}

func SetScreenSize(width, height int) {
	ui.CurrentUI().SetScreenSize(width, height)
}

func SetScreenScale(scale int) {
	ui.CurrentUI().SetScreenScale(scale)
}

func Render() {
	runtime.LockOSThread()
	// TODO: Implement this
	/*select {
	case <-workAvailable:
		DoWork()
	case <-done:
		return
	}*/
}

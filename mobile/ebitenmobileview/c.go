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

// #cgo ios LDFLAGS: -framework UIKit -framework GLKit -framework QuartzCore -framework OpenGLES
//
// #include <stdint.h>
import "C"

import (
	"math"
	"runtime"
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

	running bool

	// m is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	m sync.Mutex
}

func SetGame(game game) {
	theState.m.Lock()
	defer theState.m.Unlock()

	theState.game = game
}

//export ebitenLayout
func ebitenLayout(viewWidth, viewHeight C.int, x, y, width, height *C.int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	if theState.game == nil {
		panic("ebitenmobileview: SetGame must be called before ebitenLayout")
	}

	w, h := theState.game.Layout(int(viewWidth), int(viewHeight))
	scaleX := float64(viewWidth) / float64(w)
	scaleY := float64(viewHeight) / float64(h)
	scale := math.Min(scaleX, scaleY)

	*width = C.int(math.Ceil(float64(w) * scale))
	*height = C.int(math.Ceil(float64(h) * scale))
	*x = (viewWidth - *width) / 2
	*y = (viewHeight - *height) / 2

	if !theState.running {
		start(theState.game.Update, w, h, scale)
		theState.running = true
	}
	// TODO: call SetScreenSize
}

//export ebitenUpdate
func ebitenUpdate() *C.char {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()

	if err := update(); err != nil {
		// TODO: When to free cstr?
		cstr := C.CString(err.Error())
		return cstr
	}
	return nil
}

//export ebitenUpdateTouchesOnIOS
func ebitenUpdateTouchesOnIOS(phase C.int, ptr C.uintptr_t, x, y C.int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	updateTouchesOnIOSImpl(int(phase), int64(ptr), int(x), int(y))
}

//export ebitenUpdateTouchesOnAndroid
func ebitenUpdateTouchesOnAndroid(action C.int, id C.int, x, y C.int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	updateTouchesOnAndroid(int(action), int(id), int(x), int(y))
}

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

// Package ebitenmobileview offers functions for OpenGL/Metal view of mobiles.
//
// The functions are not intended for public usages.
// There is no guarantee of backward compatibility.
package ebitenmobileview

import (
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten"
)

var theState state

type state struct {
	updateFunc   func(*ebiten.Image) error
	screenWidth  int
	screenHeight int

	// m is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	m sync.Mutex
}

func Run(scale float64) {
	theState.m.Lock()
	defer theState.m.Unlock()

	if theState.updateFunc == nil {
		panic("ebitenmobileview: SetUpdateFunc must be called before Run")
	}
	start(theState.updateFunc, theState.screenWidth, theState.screenHeight, scale)
}

func Update() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()

	return update()
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	updateTouchesOnAndroid(action, id, x, y)
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	updateTouchesOnIOSImpl(phase, ptr, x, y)
}

func ScreenWidth() int {
	theState.m.Lock()
	defer theState.m.Unlock()
	return theState.screenWidth
}

func ScreenHeight() int {
	theState.m.Lock()
	defer theState.m.Unlock()
	return theState.screenHeight
}

func Set(updateFunc func(*ebiten.Image) error, screenWidth, screenHeight int) {
	theState.m.Lock()
	defer theState.m.Unlock()

	theState.updateFunc = updateFunc
	theState.screenWidth = screenWidth
	theState.screenHeight = screenHeight
}

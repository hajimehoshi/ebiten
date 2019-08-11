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

var (
	// mobileMutex is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	mobileMutex sync.Mutex
)

func Run(width, height int, scale float64) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	if updateFunc == nil {
		panic("ebitenmobileview: SetUpdateFunc must be called before Run")
	}
	start(updateFunc, width, height, scale)
}

func Update() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	return update()
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	updateTouchesOnAndroid(action, id, x, y)
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	updateTouchesOnIOSImpl(phase, ptr, x, y)
}

var updateFunc func(*ebiten.Image) error

func SetUpdateFunc(f func(*ebiten.Image) error) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	updateFunc = f
}

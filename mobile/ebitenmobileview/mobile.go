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

// Package ebitenmobileview offers functions for OpenGL/Metal view of mobiles.
//
// The functions are not intended for public usages.
// There is no guarantee of backward compatibility.
package ebitenmobileview

// #cgo ios LDFLAGS: -framework UIKit -framework GLKit -framework QuartzCore -framework OpenGLES
//
// #include <stdint.h>
import "C"

import (
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/uidriver/mobile"
)

var theState state

type state struct {
	started bool

	// m is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	m sync.Mutex
}

func (s *state) isRunning() bool {
	return s.started
}

func SetGame(game ebiten.Game) {
	theState.m.Lock()
	defer theState.m.Unlock()

	if theState.started {
		panic("ebitenmobileview: SetGame cannot be called twice or more")
	}
	ebiten.RunGameWithoutMainLoop(game)
	theState.started = true
}

func Layout(viewWidth, viewHeight float64) {
	theState.m.Lock()
	defer theState.m.Unlock()

	mobile.Get().SetOutsideSize(viewWidth, viewHeight)
}

func Update() error {
	// Lock the OS thread since graphics functions (GL) must be called on this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()
	if !theState.started {
		// start is not called yet, but as update can be called from another thread, it is OK. Just ignore
		// this.
		return nil
	}

	return mobile.Get().Update()
}

func Suspend() {
	mobile.Get().SetForeground(false)
}

func Resume() {
	mobile.Get().SetForeground(true)
}

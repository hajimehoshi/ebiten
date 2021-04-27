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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	"github.com/hajimehoshi/ebiten/v2/internal/uidriver/mobile"
)

var theState state

type state struct {
	running int32
}

func (s *state) isRunning() bool {
	return atomic.LoadInt32(&s.running) != 0
}

func (s *state) run() {
	atomic.StoreInt32(&s.running, 1)
}

func SetGame(game ebiten.Game) {
	if theState.isRunning() {
		panic("ebitenmobileview: SetGame cannot be called twice or more")
	}
	ebiten.RunGameWithoutMainLoop(game)
	theState.run()
}

func Layout(viewWidth, viewHeight float64) {
	mobile.Get().SetOutsideSize(viewWidth, viewHeight)
}

func Update() error {
	// Lock the OS thread since graphics functions (GL) must be called on this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !theState.isRunning() {
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

func OnContextLost() {
	restorable.OnContextLost()
}

func DeviceScale() float64 {
	return devicescale.GetAt(0, 0)
}

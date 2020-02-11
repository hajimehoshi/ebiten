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

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/uidriver/mobile"
)

func Layout(viewWidth, viewHeight float64) {
	theState.m.Lock()
	defer theState.m.Unlock()
	layout(viewWidth, viewHeight)
}

func layout(viewWidth, viewHeight float64) {
	if theState.game == nil {
		// It is fine to override the existing function since only the last layout result matters.
		theState.delayedLayout = func() {
			layout(viewWidth, viewHeight)
		}
		return
	}

	mobile.Get().SetOutsideSize(viewWidth, viewHeight)
	if !theState.isRunning() {
		theState.errorCh = ebiten.RunGameWithoutMainLoop(theState.game)
	}
}

func Update() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()

	return update()
}

func update() error {
	if !theState.isRunning() {
		// start is not called yet, but as update can be called from another thread, it is OK. Just ignore
		// this.
		return nil
	}

	select {
	case err := <-theState.errorCh:
		return err
	default:
	}

	mobile.Get().Update()
	return nil
}

func Suspend() {
	mobile.Get().SetForeground(false)
}

func Resume() {
	mobile.Get().SetForeground(true)
}

// Copyright 2022 The Ebitengine Authors
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

//go:build !ios && !ebitenginecbackend && !ebitencbackend
// +build !ios,!ebitenginecbackend,!ebitencbackend

package ui

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework CoreGraphics
//
// #import "CoreGraphics/CoreGraphics.h"
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type nativeInput struct {
	keyPressed map[C.CGKeyCode]bool
}

func (i *Input) updateKeys(window *glfw.Window) {
	if i.keyPressed == nil {
		i.keyPressed = map[C.CGKeyCode]bool{}
	}

	if window.GetAttrib(glfw.Focused) != glfw.True {
		for _, cgKey := range uiKeyToCGKey {
			i.keyPressed[C.CGKeyCode(cgKey)] = false
		}
		return
	}

	// Record the key states instead of calling CGEventSourceKeyState every time at IsKeyPressed.
	// There is an assumption that the key states never change during one tick.
	// Without this assumption, some functions in inpututil would not work correctly.
	for _, cgKey := range uiKeyToCGKey {
		i.keyPressed[C.CGKeyCode(cgKey)] = bool(C.CGEventSourceKeyState(C.kCGEventSourceStateCombinedSessionState, C.CGKeyCode(cgKey)))
	}
}

func (i *Input) IsKeyPressed(key Key) bool {
	if !i.ui.isRunning() {
		return false
	}

	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	k, ok := uiKeyToCGKey[key]
	if !ok {
		return false
	}
	return i.keyPressed[C.CGKeyCode(k)]
}

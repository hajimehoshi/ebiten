// Copyright 2026 The Ebitengine Authors
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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// X11InputContextOnMainThread is called from the main thread.
// The textinput package invokes X11InputContextOnMainThread to get the
// window's X input context (XIC), which is 0 when no input method is
// available.
func (u *UserInterface) X11InputContextOnMainThread() uintptr {
	b, ok := u.runningBackend().(*glfwBackend)
	if !ok {
		return 0
	}
	ic, err := b.window.GetX11InputContext()
	if err != nil {
		return 0
	}
	return ic
}

// ResetX11InputContextOnMainThread discards the composition the input method
// holds for the window.
//
// ResetX11InputContextOnMainThread must be called from the main thread.
func (u *UserInterface) ResetX11InputContextOnMainThread() {
	b, ok := u.runningBackend().(*glfwBackend)
	if !ok {
		return
	}
	_ = b.window.ResetInputContext()
}

// SetX11TextInputHandlersOnMainThread registers the handlers the textinput
// package receives input method events with. onPreedit reports a composition
// update, where selStartInBytes and selEndInBytes delimit the highlighted part
// of text, and onCommit reports committed text. isActive is asked whether text
// inputting is in progress, which decides whether a key press waits for the
// input method to decline it. Any of them may be nil.
//
// The handlers are called from the main thread while events are processed.
//
// SetX11TextInputHandlersOnMainThread must be called from the main thread.
func (u *UserInterface) SetX11TextInputHandlersOnMainThread(onPreedit func(text string, selStartInBytes, selEndInBytes int), onCommit func(text string), isActive func() bool) {
	b, ok := u.runningBackend().(*glfwBackend)
	if !ok {
		return
	}
	w := b.window
	if onPreedit != nil {
		_, _ = w.SetPreeditCallback(func(_ *glfw.Window, text string, selStartInBytes, selEndInBytes int) {
			onPreedit(text, selStartInBytes, selEndInBytes)
		})
	}
	if onCommit != nil {
		_, _ = w.SetTextInputCallback(func(_ *glfw.Window, text string) {
			onCommit(text)
		})
	}
	if isActive != nil {
		_, _ = w.SetTextInputActiveCallback(func(_ *glfw.Window) bool {
			return isActive()
		})
	}
}

// Copyright 2021 The Ebiten Authors
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

//go:build !js && !windows
// +build !js,!windows

package glfw

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	charModsCallbacks        = map[CharModsCallback]glfw.CharModsCallback{}
	closeCallbacks           = map[CloseCallback]glfw.CloseCallback{}
	framebufferSizeCallbacks = map[FramebufferSizeCallback]glfw.FramebufferSizeCallback{}
	scrollCallbacks          = map[ScrollCallback]glfw.ScrollCallback{}
	sizeCallbacks            = map[SizeCallback]glfw.SizeCallback{}
)

func ToCharModsCallback(cb func(window *Window, char rune, mods ModifierKey)) CharModsCallback {
	if cb == nil {
		return 0
	}
	id := CharModsCallback(len(charModsCallbacks) + 1)
	var gcb glfw.CharModsCallback = func(window *glfw.Window, char rune, mods glfw.ModifierKey) {
		cb(theWindows.get(window), char, ModifierKey(mods))
	}
	charModsCallbacks[id] = gcb
	return id
}

func ToCloseCallback(cb func(window *Window)) CloseCallback {
	if cb == nil {
		return 0
	}
	id := CloseCallback(len(closeCallbacks) + 1)
	var gcb glfw.CloseCallback = func(window *glfw.Window) {
		cb(theWindows.get(window))
	}
	closeCallbacks[id] = gcb
	return id
}

func ToFramebufferSizeCallback(cb func(window *Window, width int, height int)) FramebufferSizeCallback {
	if cb == nil {
		return 0
	}
	id := FramebufferSizeCallback(len(framebufferSizeCallbacks) + 1)
	var gcb glfw.FramebufferSizeCallback = func(window *glfw.Window, width int, height int) {
		cb(theWindows.get(window), width, height)
	}
	framebufferSizeCallbacks[id] = gcb
	return id
}

func ToScrollCallback(cb func(window *Window, xoff float64, yoff float64)) ScrollCallback {
	if cb == nil {
		return 0
	}
	id := ScrollCallback(len(scrollCallbacks) + 1)
	var gcb glfw.ScrollCallback = func(window *glfw.Window, xoff float64, yoff float64) {
		cb(theWindows.get(window), xoff, yoff)
	}
	scrollCallbacks[id] = gcb
	return id
}

func ToSizeCallback(cb func(window *Window, width int, height int)) SizeCallback {
	if cb == nil {
		return 0
	}
	id := SizeCallback(len(sizeCallbacks) + 1)
	var gcb glfw.SizeCallback = func(window *glfw.Window, width, height int) {
		cb(theWindows.get(window), width, height)
	}
	sizeCallbacks[id] = gcb
	return id
}

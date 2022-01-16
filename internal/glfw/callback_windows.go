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

package glfw

import (
	"golang.org/x/sys/windows"
)

func ToCharModsCallback(cb func(window *Window, char rune, mods ModifierKey)) CharModsCallback {
	if cb == nil {
		return 0
	}
	return CharModsCallback(windows.NewCallbackCDecl(func(window uintptr, char rune, mods ModifierKey) uintptr {
		cb(theGLFWWindows.get(window), char, mods)
		return 0
	}))
}

func ToCloseCallback(cb func(window *Window)) CloseCallback {
	if cb == nil {
		return 0
	}
	return CloseCallback(windows.NewCallbackCDecl(func(window uintptr) uintptr {
		cb(theGLFWWindows.get(window))
		return 0
	}))
}

func ToFramebufferSizeCallback(cb func(window *Window, width int, height int)) FramebufferSizeCallback {
	if cb == nil {
		return 0
	}
	return FramebufferSizeCallback(windows.NewCallbackCDecl(func(window uintptr, width int, height int) uintptr {
		cb(theGLFWWindows.get(window), width, height)
		return 0
	}))
}

func ToScrollCallback(cb func(window *Window, xoff float64, yoff float64)) ScrollCallback {
	if cb == nil {
		return 0
	}
	return ScrollCallback(windows.NewCallbackCDecl(func(window uintptr, xoff *float64, yoff *float64) uintptr {
		// xoff and yoff were originally float64, but there is no good way to pass them on 32bit
		// machines via NewCallback. We've fixed GLFW side to use pointer values.
		cb(theGLFWWindows.get(window), *xoff, *yoff)
		return 0
	}))
}

func ToSizeCallback(cb func(window *Window, width int, height int)) SizeCallback {
	if cb == nil {
		return 0
	}
	return SizeCallback(windows.NewCallbackCDecl(func(window uintptr, width int, height int) uintptr {
		cb(theGLFWWindows.get(window), width, height)
		return 0
	}))
}

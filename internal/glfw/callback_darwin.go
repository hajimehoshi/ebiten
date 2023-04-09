// Copyright 2023 The Ebitengine Authors
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
	"unsafe"

	"golang.org/x/sys/unix"
)

func ToCharModsCallback(cb func(window *Window, char rune, mods ModifierKey)) CharModsCallback {
	if cb == nil {
		return nil
	}
	return func(window uintptr, char rune, mods ModifierKey) {
		cb(theGLFWWindows.get(window), char, mods)
	}
}

func ToCloseCallback(cb func(window *Window)) CloseCallback {
	if cb == nil {
		return nil
	}
	return func(window uintptr) {
		cb(theGLFWWindows.get(window))
	}
}

func ToFramebufferSizeCallback(cb func(window *Window, width int, height int)) FramebufferSizeCallback {
	if cb == nil {
		return nil
	}
	return func(window uintptr, width int, height int) {
		cb(theGLFWWindows.get(window), width, height)
	}
}

func ToMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) MonitorCallback {
	if cb == nil {
		return nil
	}
	return func(monitor uintptr, event PeripheralEvent) {
		cb(&Monitor{m: monitor}, event)
	}
}

func ToScrollCallback(cb func(window *Window, xoff float64, yoff float64)) ScrollCallback {
	if cb == nil {
		return nil
	}
	return func(window uintptr, xoff float64, yoff float64) {
		cb(theGLFWWindows.get(window), xoff, yoff)
	}
}

func ToDropCallback(cb func(window *Window, names []string)) DropCallback {
	if cb == nil {
		return nil
	}
	return func(window uintptr, count int, path **byte) {
		strs := make([]string, count)
		names := unsafe.Slice(path, count)
		for i := range strs {
			n := names[i]
			if n == nil {
				break
			}
			strs[i] = unix.BytePtrToString(n)
		}
		cb(theGLFWWindows.get(window), strs)
	}
}

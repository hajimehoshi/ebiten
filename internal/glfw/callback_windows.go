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

//go:build windows

package glfw

func ToCharModsCallback(cb func(window *Window, char rune, mods ModifierKey)) CharModsCallback {
	return cb
}

func ToCloseCallback(cb func(window *Window)) CloseCallback {
	return cb
}

func ToDropCallback(cb func(window *Window, names []string)) DropCallback {
	return cb
}

func ToFramebufferSizeCallback(cb func(window *Window, width int, height int)) FramebufferSizeCallback {
	return cb
}

func ToMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) MonitorCallback {
	return cb
}

func ToScrollCallback(cb func(window *Window, xoff float64, yoff float64)) ScrollCallback {
	return cb
}

func ToSizeCallback(cb func(window *Window, width int, height int)) SizeCallback {
	return cb
}

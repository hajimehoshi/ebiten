// Copyright 2022 The Ebiten Authors
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

//go:build android || ios || js || ebitencbackend
// +build android ios js ebitencbackend

package ui

import (
	"image"
)

type Window struct{}

func (*Window) IsDecorated() bool {
	return false
}

func (*Window) SetDecorated(decorated bool) {
}

func (*Window) ResizingMode() WindowResizingMode {
	return WindowResizingModeDisabled
}

func (*Window) SetResizingMode(mode WindowResizingMode) {
}

func (*Window) Position() (int, int) {
	return 0, 0
}

func (*Window) SetPosition(x, y int) {
}

func (*Window) Size() (int, int) {
	return 0, 0
}

func (*Window) SetSize(width, height int) {
}

func (*Window) SizeLimits() (minw, minh, maxw, maxh int) {
	return -1, -1, -1, -1
}

func (*Window) SetSizeLimits(minw, minh, maxw, maxh int) {
}

func (*Window) IsFloating() bool {
	return false
}

func (*Window) SetFloating(floating bool) {
}

func (*Window) Maximize() {
}

func (*Window) IsMaximized() bool {
	return false
}

func (*Window) Minimize() {
}

func (*Window) IsMinimized() bool {
	return false
}

func (*Window) SetIcon(iconImages []image.Image) {
}

func (*Window) SetTitle(title string) {
}

func (*Window) Restore() {
}

func (*Window) IsBeingClosed() bool {
	return false
}

func (*Window) SetClosingHandled(handled bool) {
}

func (*Window) IsClosingHandled() bool {
	return false
}

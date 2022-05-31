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

package ui

import (
	"image"
)

type Window interface {
	IsDecorated() bool
	SetDecorated(decorated bool)
	ResizingMode() WindowResizingMode
	SetResizingMode(mode WindowResizingMode)
	Position() (int, int)
	SetPosition(x, y int)
	Size() (int, int)
	SetSize(width, height int)
	SizeLimits() (minw, minh, maxw, maxh int)
	SetSizeLimits(minw, minh, maxw, maxh int)
	IsFloating() bool
	SetFloating(floating bool)
	Maximize()
	IsMaximized() bool
	Minimize()
	IsMinimized() bool
	SetIcon(iconImages []image.Image)
	SetTitle(title string)
	Restore()
	IsBeingClosed() bool
	SetClosingHandled(handled bool)
	IsClosingHandled() bool
}

type nullWindow struct{}

func (*nullWindow) IsDecorated() bool {
	return false
}

func (*nullWindow) SetDecorated(decorated bool) {
}

func (*nullWindow) ResizingMode() WindowResizingMode {
	return WindowResizingModeDisabled
}

func (*nullWindow) SetResizingMode(mode WindowResizingMode) {
}

func (*nullWindow) Position() (int, int) {
	return 0, 0
}

func (*nullWindow) SetPosition(x, y int) {
}

func (*nullWindow) Size() (int, int) {
	return 0, 0
}

func (*nullWindow) SetSize(width, height int) {
}

func (*nullWindow) SizeLimits() (minw, minh, maxw, maxh int) {
	return -1, -1, -1, -1
}

func (*nullWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
}

func (*nullWindow) IsFloating() bool {
	return false
}

func (*nullWindow) SetFloating(floating bool) {
}

func (*nullWindow) Maximize() {
}

func (*nullWindow) IsMaximized() bool {
	return false
}

func (*nullWindow) Minimize() {
}

func (*nullWindow) IsMinimized() bool {
	return false
}

func (*nullWindow) SetIcon(iconImages []image.Image) {
}

func (*nullWindow) SetTitle(title string) {
}

func (*nullWindow) Restore() {
}

func (*nullWindow) IsBeingClosed() bool {
	return false
}

func (*nullWindow) SetClosingHandled(handled bool) {
}

func (*nullWindow) IsClosingHandled() bool {
	return false
}

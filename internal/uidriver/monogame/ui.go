// Copyright 2020 The Ebiten Authors
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

// +build js

package monogame

import (
	"github.com/hajimehoshi/ebiten/internal/driver"
)

type UI struct {
}

var theUI = &UI{}

func Get() *UI {
	return theUI
}

func (*UI) Run(context driver.UIContext) error {
	return nil
}

func (*UI) RunWithoutMainLoop(context driver.UIContext) {
	panic("monogame: RunWithoutMainLoop is not implemented")
}

func (*UI) DeviceScaleFactor() float64 {
	return 1
}

func (*UI) IsFocused() bool {
	return true
}

func (*UI) ScreenSizeInFullscreen() (int, int) {
	// TODO: Implement this
	return 0, 0
}

func (*UI) CursorMode() driver.CursorMode {
	return driver.CursorModeVisible
}

func (*UI) SetCursorMode(mode driver.CursorMode) {
	// TODO: Implement this
}

func (*UI) IsFullscreen() bool {
	// TODO: Implement this
	return false
}

func (*UI) SetFullscreen(fullscreen bool) {
	// TODO: Implement this
}

func (*UI) IsRunnableOnUnfocused() bool {
	// TODO: Implement this
	return false
}

func (*UI) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	// TODO: Implement this
}

func (*UI) IsVsyncEnabled() bool {
	// TODO: Implement this
	return true
}

func (*UI) SetVsyncEnabled(enabled bool) {
	// TODO: Implement this
}

func (*UI) IsScreenTransparent() bool {
	return false
}

func (*UI) SetScreenTransparent(transparent bool) {
	panic("monogame: SetScreenTransparent is not implemented")
}

func (*UI) Input() driver.Input {
	return nil
}

func (*UI) Window() driver.Window {
	return nil
}

func (*UI) Graphics() driver.Graphics {
	return nil
}

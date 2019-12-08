// Copyright 2019 The Ebiten Authors
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

package driver

import (
	"errors"
	"image"
)

type UIContext interface {
	SetSize(width, height int, scale float64)
	Update(afterFrameUpdate func()) error
}

// RegularTermination represents a regular termination.
// Run can return this error, and if this error is received,
// the game loop should be terminated as soon as possible.
var RegularTermination = errors.New("regular termination")

type UI interface {
	Run(width, height int, scale float64, title string, context UIContext, graphics Graphics) error
	RunWithoutMainLoop(width, height int, scale float64, title string, context UIContext, graphics Graphics) <-chan error

	DeviceScaleFactor() float64
	IsCursorVisible() bool
	IsFullscreen() bool
	IsRunnableInBackground() bool
	IsVsyncEnabled() bool
	IsWindowDecorated() bool
	IsWindowResizable() bool
	ScreenPadding() (x0, y0, x1, y1 float64)
	ScreenScale() float64
	ScreenSizeInFullscreen() (int, int)
	WindowPosition() (int, int)
	IsScreenTransparent() bool

	SetCursorVisible(visible bool)
	SetFullscreen(fullscreen bool)
	SetRunnableInBackground(runnableInBackground bool)
	SetScreenScale(scale float64)
	SetScreenSize(width, height int)
	SetVsyncEnabled(enabled bool)
	SetWindowDecorated(decorated bool)
	SetWindowIcon(iconImages []image.Image)
	SetWindowResizable(resizable bool)
	SetWindowTitle(title string)
	SetWindowPosition(x, y int)
	SetScreenTransparent(transparent bool)

	Input() Input
}

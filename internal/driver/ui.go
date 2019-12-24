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
	Update(afterFrameUpdate func()) error
	Layout(outsideWidth, outsideHeight float64)
	AdjustPosition(x, y float64) (float64, float64)
}

// RegularTermination represents a regular termination.
// Run can return this error, and if this error is received,
// the game loop should be terminated as soon as possible.
var RegularTermination = errors.New("regular termination")

type UI interface {
	Run(context UIContext, graphics Graphics) error
	RunWithoutMainLoop(width, height int, scale float64, title string, context UIContext, graphics Graphics) <-chan error

	DeviceScaleFactor() float64
	CursorMode() CursorMode
	IsFullscreen() bool
	IsRunnableInBackground() bool
	IsVsyncEnabled() bool
	ScreenSizeInFullscreen() (int, int)
	IsScreenTransparent() bool
	MonitorPosition() (int, int)

	SetCursorMode(mode CursorMode)
	SetFullscreen(fullscreen bool)
	SetRunnableInBackground(runnableInBackground bool)
	SetVsyncEnabled(enabled bool)
	SetScreenTransparent(transparent bool)

	Input() Input
	Window() Window
}

type Window interface {
	IsDecorated() bool
	SetDecorated(decorated bool)

	IsResizable() bool
	SetResizable(resizable bool)

	Position() (int, int)
	SetPosition(x, y int)

	Size() (int, int)
	SetSize(width, height int)

	SetIcon(iconImages []image.Image)
	SetTitle(title string)
}

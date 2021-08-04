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
	UpdateFrame() error
	ForceUpdateFrame() error
	Layout(outsideWidth, outsideHeight float64)

	// AdjustPosition can be called from a different goroutine from Update's or Layout's.
	AdjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64)
}

// RegularTermination represents a regular termination.
// Run can return this error, and if this error is received,
// the game loop should be terminated as soon as possible.
var RegularTermination = errors.New("regular termination")

type UI interface {
	Run(context UIContext) error
	RunWithoutMainLoop(context UIContext)

	DeviceScaleFactor() float64
	IsFocused() bool
	ScreenSizeInFullscreen() (int, int)
	ResetForFrame()

	CursorMode() CursorMode
	SetCursorMode(mode CursorMode)

	CursorShape() CursorShape
	SetCursorShape(shape CursorShape)

	IsFullscreen() bool
	SetFullscreen(fullscreen bool)

	IsRunnableOnUnfocused() bool
	SetRunnableOnUnfocused(runnableOnUnfocused bool)

	FPSMode() FPSMode
	SetFPSMode(mode FPSMode)
	ScheduleFrame()

	IsScreenTransparent() bool
	SetScreenTransparent(transparent bool)
	SetInitFocused(focused bool)

	Input() Input
	Window() Window
	Graphics() Graphics
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

type FPSMode int

const (
	FPSModeVsyncOn FPSMode = iota
	FPSModeVsyncOffMaximum
	FPSModeVsyncOffMinimum
)

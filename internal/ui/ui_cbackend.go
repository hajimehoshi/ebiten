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

//go:build ebitencbackend
// +build ebitencbackend

package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/cbackend"
)

const deviceScaleFactor = 1

func init() {
	runtime.LockOSThread()
}

type UserInterface struct {
	context *contextImpl
	input   Input
}

var theUserInterface UserInterface

func Get() *UserInterface {
	return &theUserInterface
}

func (u *UserInterface) Run(game Game) error {
	u.context = newContextImpl(game)
	cbackend.InitializeGame()
	for {
		w, h := cbackend.ScreenSize()
		u.context.layout(float64(w), float64(h))

		cbackend.BeginFrame()
		u.input.update(u.context)

		if err := u.context.updateFrame(deviceScaleFactor); err != nil {
			return err
		}

		cbackend.EndFrame()
	}
}

func (*UserInterface) DeviceScaleFactor() float64 {
	return deviceScaleFactor
}

func (*UserInterface) IsFocused() bool {
	return true
}

func (*UserInterface) ScreenSizeInFullscreen() (int, int) {
	return 0, 0
}

func (*UserInterface) resetForTick() {
}

func (*UserInterface) CursorMode() CursorMode {
	return CursorModeHidden
}

func (*UserInterface) SetCursorMode(mode CursorMode) {
}

func (*UserInterface) CursorShape() CursorShape {
	return CursorShapeDefault
}

func (*UserInterface) SetCursorShape(shape CursorShape) {
}

func (*UserInterface) IsFullscreen() bool {
	return false
}

func (*UserInterface) SetFullscreen(fullscreen bool) {
}

func (*UserInterface) IsRunnableOnUnfocused() bool {
	return false
}

func (*UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
}

func (*UserInterface) SetFPSMode(mode FPSModeType) {
}

func (*UserInterface) ScheduleFrame() {
}

func (*UserInterface) IsScreenTransparent() bool {
	return false
}

func (*UserInterface) SetScreenTransparent(transparent bool) {
}

func (*UserInterface) SetInitFocused(focused bool) {
}

func (*UserInterface) Input() *Input {
	return &theUserInterface.input
}

func (*UserInterface) Window() *Window {
	return &Window{}
}

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
	input Input
}

var theUserInterface UserInterface

func Get() *UserInterface {
	return &theUserInterface
}

func (u *UserInterface) Run(context Context) error {
	cbackend.InitializeGame()
	for {
		w, h := cbackend.ScreenSize()
		context.Layout(float64(w), float64(h))

		cbackend.BeginFrame()
		u.input.update(context)

		if err := context.UpdateFrame(); err != nil {
			return err
		}

		cbackend.EndFrame()
	}
}

func (*UserInterface) RunWithoutMainLoop(context Context) {
	panic("ui: RunWithoutMainLoop is not implemented")
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

func (*UserInterface) ResetForFrame() {
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

func (*UserInterface) FPSMode() FPSMode {
	return FPSModeVsyncOn
}

func (*UserInterface) SetFPSMode(mode FPSMode) {
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

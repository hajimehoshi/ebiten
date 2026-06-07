// Copyright 2026 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// uiBackend is the platform UI implementation for the desktop build.
type uiBackend interface {
	init() error
	run(game Game, options *RunOptions) error
	readInputState(inputState *InputState)
	updateInputStateForFrame(deviceScaleFactor float64) error
	updateIconIfNeeded() error
	IsFocused() bool
	IsFullscreen() bool
	SetFullscreen(fullscreen bool)
	IsRunnableOnUnfocused() bool
	SetRunnableOnUnfocused(runnableOnUnfocused bool)
	FPSMode() FPSModeType
	SetFPSMode(mode FPSModeType)
	ScheduleFrame()
	CursorMode() CursorMode
	SetCursorMode(mode CursorMode)
	CursorShape() CursorShape
	SetCursorShape(shape CursorShape)
	Window() Window
	Monitor() *Monitor
	AppendMonitors(monitors []*Monitor) []*Monitor
	RunOnMainThread(f func())
	KeyName(key Key) string
}

var _ uiBackend = (*glfwBackend)(nil)

type userInterfaceImpl struct {
	backend uiBackend

	graphicsDriver graphicsdriver.Graphics
	context        *context
}

func newGlfwBackend(u *UserInterface) *glfwBackend {
	return &glfwBackend{UserInterface: u}
}

func (u *UserInterface) init() error {
	u.backend = newGlfwBackend(u)
	return u.backend.init()
}

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	return u.backend.run(game, options)
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.backend.readInputState(inputState)
}

func (u *UserInterface) updateInputStateForFrame(deviceScaleFactor float64) error {
	return u.backend.updateInputStateForFrame(deviceScaleFactor)
}

func (u *UserInterface) updateIconIfNeeded() error {
	return u.backend.updateIconIfNeeded()
}

func (u *UserInterface) IsFocused() bool {
	return u.backend.IsFocused()
}

func (u *UserInterface) IsFullscreen() bool {
	return u.backend.IsFullscreen()
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	u.backend.SetFullscreen(fullscreen)
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.backend.IsRunnableOnUnfocused()
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.backend.SetRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *UserInterface) FPSMode() FPSModeType {
	return u.backend.FPSMode()
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	u.backend.SetFPSMode(mode)
}

func (u *UserInterface) ScheduleFrame() {
	u.backend.ScheduleFrame()
}

func (u *UserInterface) CursorMode() CursorMode {
	return u.backend.CursorMode()
}

func (u *UserInterface) SetCursorMode(mode CursorMode) {
	u.backend.SetCursorMode(mode)
}

func (u *UserInterface) CursorShape() CursorShape {
	return u.backend.CursorShape()
}

func (u *UserInterface) SetCursorShape(shape CursorShape) {
	u.backend.SetCursorShape(shape)
}

func (u *UserInterface) Window() Window {
	return u.backend.Window()
}

func (u *UserInterface) Monitor() *Monitor {
	return u.backend.Monitor()
}

func (u *UserInterface) AppendMonitors(monitors []*Monitor) []*Monitor {
	return u.backend.AppendMonitors(monitors)
}

func (u *UserInterface) RunOnMainThread(f func()) {
	u.backend.RunOnMainThread(f)
}

func (u *UserInterface) KeyName(key Key) string {
	return u.backend.KeyName(key)
}

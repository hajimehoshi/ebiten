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

//go:build nintendosdk

package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/nintendosdk"
)

type graphicsDriverCreatorImpl struct{}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	graphics, err := g.newOpenGL()
	return graphics, GraphicsLibraryOpenGL, err
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, nil
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, nil
}

const deviceScaleFactor = 1

func init() {
	runtime.LockOSThread()
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	context       *context
	inputState    InputState
	nativeTouches []nintendosdk.Touch
}

func (u *userInterfaceImpl) Run(game Game, options *RunOptions) error {
	u.context = newContext(game)
	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	nintendosdk.InitializeGame()
	for {
		nintendosdk.BeginFrame()
		gamepad.Update()
		u.updateInputState()

		w, h := nintendosdk.ScreenSize()
		if err := u.context.updateFrame(u.graphicsDriver, float64(w), float64(h), deviceScaleFactor, u); err != nil {
			return err
		}

		nintendosdk.EndFrame()
	}
}

func (*userInterfaceImpl) DeviceScaleFactor() float64 {
	return deviceScaleFactor
}

func (*userInterfaceImpl) IsFocused() bool {
	return true
}

func (*userInterfaceImpl) ScreenSizeInFullscreen() (int, int) {
	return 0, 0
}

func (u *userInterfaceImpl) readInputState(inputState *InputState) {
	*inputState = u.inputState
}

func (u *userInterfaceImpl) resetForTick() {
	u.inputState.resetForTick()
}

func (*userInterfaceImpl) CursorMode() CursorMode {
	return CursorModeHidden
}

func (*userInterfaceImpl) SetCursorMode(mode CursorMode) {
}

func (*userInterfaceImpl) CursorShape() CursorShape {
	return CursorShapeDefault
}

func (*userInterfaceImpl) SetCursorShape(shape CursorShape) {
}

func (*userInterfaceImpl) IsFullscreen() bool {
	return false
}

func (*userInterfaceImpl) SetFullscreen(fullscreen bool) {
}

func (*userInterfaceImpl) IsRunnableOnUnfocused() bool {
	return false
}

func (*userInterfaceImpl) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
}

func (*userInterfaceImpl) SetFPSMode(mode FPSModeType) {
}

func (*userInterfaceImpl) ScheduleFrame() {
}

func (*userInterfaceImpl) Window() Window {
	return &nullWindow{}
}

func (u *userInterfaceImpl) beginFrame() {
}

func (u *userInterfaceImpl) endFrame() {
}

func IsScreenTransparentAvailable() bool {
	return false
}

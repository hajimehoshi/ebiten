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

// #include "init_nintendosdk.h"
// #include "input_nintendosdk.h"
import "C"

import (
	"errors"
	"image"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
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
	return nil, errors.New("ui: DirectX is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: Metal is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
}

const deviceScaleFactor = 1

func init() {
	runtime.LockOSThread()
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	context       *context
	inputState    InputState
	nativeTouches []C.struct_Touch

	egl egl

	m sync.Mutex
}

func (u *UserInterface) init() error {
	return nil
}

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	return u.run(game, options)
}

func (u *UserInterface) initOnMainThread(options *RunOptions) error {
	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.setGraphicsLibrary(lib)

	n := C.ebitengine_Initialize()
	if err := u.egl.init(n); err != nil {
		return err
	}

	initializeProfiler()

	return nil
}

func (u *UserInterface) loopGame() error {
	u.renderThread.Call(func() {
		u.egl.makeContextCurrent()
	})

	for {
		recordProfilerHeartbeat()

		if err := u.context.updateFrame(u.graphicsDriver, float64(C.kScreenWidth), float64(C.kScreenHeight), deviceScaleFactor, u, func() {
			u.egl.swapBuffers()
		}); err != nil {
			return err
		}
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

func (u *UserInterface) readInputState(inputState *InputState) {
	u.m.Lock()
	defer u.m.Unlock()
	u.inputState.copyAndReset(inputState)
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

func (*UserInterface) FPSMode() FPSModeType {
	return FPSModeVsyncOn
}

func (*UserInterface) SetFPSMode(mode FPSModeType) {
}

func (*UserInterface) ScheduleFrame() {
}

func (*UserInterface) Window() Window {
	return &nullWindow{}
}

func (u *UserInterface) updateIconIfNeeded() error {
	return nil
}

type Monitor struct{}

var theMonitor = &Monitor{}

func (m *Monitor) Bounds() image.Rectangle {
	// TODO: This should return the available viewport dimensions.
	return image.Rectangle{}
}

func (m *Monitor) Name() string {
	return ""
}

func (u *UserInterface) AppendMonitors(mons []*Monitor) []*Monitor {
	return append(mons, theMonitor)
}

func (u *UserInterface) Monitor() *Monitor {
	return theMonitor
}

func IsScreenTransparentAvailable() bool {
	return false
}

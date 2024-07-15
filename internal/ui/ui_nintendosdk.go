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
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	nativeWindow C.NativeWindowType
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	graphics, err := g.newOpenGL()
	return graphics, GraphicsLibraryOpenGL, err
}

func (g *graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics(uintptr(g.nativeWindow))
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

func init() {
	runtime.LockOSThread()
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	context       *context
	inputState    InputState
	nativeTouches []C.struct_Touch

	m sync.Mutex
}

func (u *UserInterface) init() error {
	return nil
}

func (u *UserInterface) initOnMainThread(options *RunOptions) error {
	u.setRunning(true)

	n := C.ebitengine_Initialize()
	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		nativeWindow: n,
	}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.setGraphicsLibrary(lib)

	initializeProfiler()

	return nil
}

func (u *UserInterface) loopGame() error {
	for {
		recordProfilerHeartbeat()

		if err := u.context.updateFrame(u.graphicsDriver, float64(C.kScreenWidth), float64(C.kScreenHeight), theMonitor.DeviceScaleFactor(), u); err != nil {
			return err
		}
	}
}

func (*UserInterface) IsFocused() bool {
	return true
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

func (m *Monitor) Name() string {
	return ""
}

func (m *Monitor) DeviceScaleFactor() float64 {
	return 1
}

func (m *Monitor) Size() (int, int) {
	return int(C.kScreenWidth), int(C.kScreenHeight)
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

func dipToNativePixels(x float64, scale float64) float64 {
	return x
}

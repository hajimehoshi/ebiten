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
	stdcontext "context"
	"image"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
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
	nativeTouches []C.struct_Touch

	egl egl

	mainThread   *thread.OSThread
	renderThread *thread.OSThread

	m sync.Mutex
}

func (u *userInterfaceImpl) Run(game Game, options *RunOptions) error {
	u.context = newContext(game)
	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g

	n := C.ebitengine_Initialize()
	if err := u.egl.init(n); err != nil {
		return err
	}

	initializeProfiler()

	u.mainThread = thread.NewOSThread()
	u.renderThread = thread.NewOSThread()
	graphicscommand.SetRenderThread(u.renderThread)

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the render thread.
	wg.Go(func() error {
		defer cancel()
		_ = u.renderThread.Loop(ctx)
		return nil
	})

	// Run the game thread.
	wg.Go(func() error {
		defer cancel()

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
	})

	// Run the main thread.
	_ = u.mainThread.Loop(ctx)
	if err := wg.Wait(); err != nil {
		return err
	}
	return nil
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
	u.m.Lock()
	defer u.m.Unlock()
	u.inputState.copyAndReset(inputState)
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

type Monitor struct{}

var theMonitor = &Monitor{}

func (m *Monitor) Bounds() image.Rectangle {
	// TODO: This should return the available viewport dimensions.
	return image.Rectangle{}
}

func (m *Monitor) Name() string {
	return ""
}

func (u *userInterfaceImpl) AppendMonitors(mons []*Monitor) []*Monitor {
	return append(mons, theMonitor)
}

func (u *userInterfaceImpl) Monitor() *Monitor {
	return theMonitor
}

func (u *userInterfaceImpl) beginFrame() {
}

func (u *userInterfaceImpl) endFrame() {
}

func (u *userInterfaceImpl) updateIconIfNeeded() error {
	return nil
}

func IsScreenTransparentAvailable() bool {
	return false
}

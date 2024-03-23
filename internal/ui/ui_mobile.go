// Copyright 2016 Hajime Hoshi
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

//go:build android || ios

package ui

import (
	stdcontext "context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

var (
	// renderCh receives when updating starts.
	renderCh = make(chan struct{})

	// renderEndCh receives when updating finishes.
	renderEndCh = make(chan struct{})
)

func (u *UserInterface) init() error {
	u.userInterfaceImpl = userInterfaceImpl{
		foreground:            1,
		graphicsLibraryInitCh: make(chan struct{}),
		errCh:                 make(chan error),

		// Give a default outside size so that the game can start without initializing them.
		outsideWidth:  640,
		outsideHeight: 480,
	}
	return nil
}

// Update is called from mobile/ebitenmobileview.
//
// Update must be called on the rendering thread.
func (u *UserInterface) Update() error {
	select {
	case err := <-u.errCh:
		return err
	default:
	}

	if !u.IsFocused() {
		return nil
	}

	if err := gamepad.Update(); err != nil {
		return err
	}

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	renderCh <- struct{}{}
	go func() {
		<-renderEndCh
		cancel()
	}()

	graphicscommand.LoopRenderThread(ctx)
	return nil
}

type userInterfaceImpl struct {
	graphicsDriver        graphicsdriver.Graphics
	graphicsLibraryInitCh chan struct{}

	outsideWidth  float64
	outsideHeight float64

	foreground int32
	errCh      chan error

	context *context

	inputState InputState
	touches    []TouchForInput

	fpsMode         int32
	renderRequester RenderRequester

	m sync.RWMutex
}

func (u *UserInterface) SetForeground(foreground bool) error {
	var v int32
	if foreground {
		v = 1
	}
	atomic.StoreInt32(&u.foreground, v)

	if foreground {
		return hook.ResumeAudio()
	} else {
		return hook.SuspendAudio()
	}
}

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	return fmt.Errorf("internal/ui: Run is not implemented for GOOS=%s", runtime.GOOS)
}

func (u *UserInterface) RunWithoutMainLoop(game Game, options *RunOptions) {
	go func() {
		if err := u.runMobile(game, options); err != nil {
			u.errCh <- err
		}
	}()
}

func (u *UserInterface) runMobile(game Game, options *RunOptions) (err error) {
	// Convert the panic to a regular error so that Java/Objective-C layer can treat this easily e.g., for
	// Crashlytics. A panic is treated as SIGABRT, and there is no way to handle this on Java/Objective-C layer
	// unfortunately.
	// TODO: Panic on other goroutines cannot be handled here.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v\n%s", r, string(debug.Stack()))
		}
	}()

	graphicscommand.SetOSThreadAsRenderThread()

	u.setRunning(true)
	defer u.setRunning(false)

	u.context = newContext(game)

	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.setGraphicsLibrary(lib)
	close(u.graphicsLibraryInitCh)

	for {
		if err := u.update(); err != nil {
			return err
		}
	}
}

// outsideSize must be called on the same goroutine as update().
func (u *UserInterface) outsideSize() (float64, float64) {
	u.m.RLock()
	defer u.m.RUnlock()

	return u.outsideWidth, u.outsideHeight
}

func (u *UserInterface) update() error {
	<-renderCh
	defer func() {
		renderEndCh <- struct{}{}
	}()

	w, h := u.outsideSize()
	if err := u.context.updateFrame(u.graphicsDriver, w, h, theMonitor.DeviceScaleFactor(), u); err != nil {
		return err
	}
	return nil
}

// SetOutsideSize is called from mobile/ebitenmobileview.
//
// SetOutsideSize is concurrent safe.
func (u *UserInterface) SetOutsideSize(outsideWidth, outsideHeight float64) {
	u.m.Lock()
	defer u.m.Unlock()
	if u.outsideWidth != outsideWidth || u.outsideHeight != outsideHeight {
		u.outsideWidth = outsideWidth
		u.outsideHeight = outsideHeight
	}
}

func (u *UserInterface) CursorMode() CursorMode {
	return CursorModeHidden
}

func (u *UserInterface) SetCursorMode(mode CursorMode) {
	// Do nothing
}

func (u *UserInterface) CursorShape() CursorShape {
	return CursorShapeDefault
}

func (u *UserInterface) SetCursorShape(shape CursorShape) {
	// Do nothing
}

func (u *UserInterface) IsFullscreen() bool {
	return false
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	// Do nothing
}

func (u *UserInterface) IsFocused() bool {
	return atomic.LoadInt32(&u.foreground) != 0
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return false
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	// Do nothing
}

func (u *UserInterface) FPSMode() FPSModeType {
	return FPSModeType(atomic.LoadInt32(&u.fpsMode))
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	atomic.StoreInt32(&u.fpsMode, int32(mode))
	u.updateExplicitRenderingModeIfNeeded(mode)
}

func (u *UserInterface) updateExplicitRenderingModeIfNeeded(fpsMode FPSModeType) {
	if u.renderRequester == nil {
		return
	}
	u.renderRequester.SetExplicitRenderingMode(fpsMode == FPSModeVsyncOffMinimum)
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.m.Lock()
	defer u.m.Unlock()
	u.inputState.copyAndReset(inputState)
}

func (u *UserInterface) Window() Window {
	return &nullWindow{}
}

type Monitor struct {
	deviceScaleFactor     float64
	deviceScaleFactorOnce sync.Once

	m sync.Mutex
}

var theMonitor = &Monitor{}

func (m *Monitor) Name() string {
	return ""
}

func (m *Monitor) DeviceScaleFactor() float64 {
	m.m.Lock()
	defer m.m.Unlock()

	// The device scale factor can be obtained after the main function starts, especially on Android.
	// Initialize this lazily.
	m.deviceScaleFactorOnce.Do(func() {
		// Assume that the device scale factor never changes on mobiles.
		m.deviceScaleFactor = deviceScaleFactorImpl()
	})
	return m.deviceScaleFactor
}

func (m *Monitor) Size() (int, int) {
	// TODO: Return a valid value.
	return 0, 0
}

func (u *UserInterface) AppendMonitors(mons []*Monitor) []*Monitor {
	return append(mons, theMonitor)
}

func (u *UserInterface) Monitor() *Monitor {
	return theMonitor
}

func (u *UserInterface) UpdateInput(keys map[Key]struct{}, runes []rune, touches []TouchForInput) {
	u.updateInputStateFromOutside(keys, runes, touches)
	if FPSModeType(atomic.LoadInt32(&u.fpsMode)) == FPSModeVsyncOffMinimum {
		u.renderRequester.RequestRenderIfNeeded()
	}
}

type RenderRequester interface {
	SetExplicitRenderingMode(explicitRendering bool)
	RequestRenderIfNeeded()
}

func (u *UserInterface) SetRenderRequester(renderRequester RenderRequester) {
	u.renderRequester = renderRequester
	u.updateExplicitRenderingModeIfNeeded(FPSModeType(atomic.LoadInt32(&u.fpsMode)))
}

func (u *UserInterface) ScheduleFrame() {
	if u.renderRequester != nil && FPSModeType(atomic.LoadInt32(&u.fpsMode)) == FPSModeVsyncOffMinimum {
		u.renderRequester.RequestRenderIfNeeded()
	}
}

func (u *UserInterface) updateIconIfNeeded() error {
	return nil
}

func IsScreenTransparentAvailable() bool {
	return false
}

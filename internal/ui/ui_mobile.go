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
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/vmguest"
)

var (
	// renderCh receives when updating starts.
	renderCh = make(chan struct{})

	// renderEndCh receives when updating finishes.
	renderEndCh = make(chan struct{})
)

func (u *UserInterface) init() error {
	u.userInterfaceImpl = userInterfaceImpl{
		graphicsLibraryInitCh: make(chan struct{}),
		errCh:                 make(chan error),
	}
	// Give a default outside size so that the game can start without initializing them.
	u.userInterfaceImpl.outsideSize.Store(&pointF{x: 640, y: 480})
	u.foreground.Store(true)
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

	if err := gamepad.Update(0, nil); err != nil {
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

type pointF struct {
	x float64
	y float64
}

type pointI struct {
	x int
	y int
}

type userInterfaceImpl struct {
	graphicsDriver        graphicsdriver.Graphics
	graphicsLibraryInitCh chan struct{}

	outsideSize atomic.Pointer[pointF]

	// surfaceSize is the size of the rendering surface, in pixels, as the platform reports it. It is
	// unset when the platform reports no size, and the size is then derived from the outside size.
	surfaceSize atomic.Pointer[pointI]

	foreground atomic.Bool
	errCh      chan error

	context *context

	inputState InputState
	touches    []TouchForInput

	fpsMode  atomic.Int32
	renderer atomic.Pointer[rendererHolder]

	// uiView is used only on iOS.
	uiView atomic.Uintptr

	mu sync.Mutex
}

func (u *UserInterface) SetForeground(foreground bool) error {
	u.foreground.Store(foreground)

	if foreground {
		return hook.ResumeAudio()
	} else {
		return hook.SuspendAudio()
	}
}

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	return fmt.Errorf("ui: Run is not implemented for GOOS=%s", runtime.GOOS)
}

func (u *UserInterface) RunWithoutMainLoop(game Game, options *RunOptions) {
	vmguest.MarkGuest(false)
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

	u.context = newContext(game, options.ScreenTransparent)

	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		colorSpace: options.ColorSpace,
	}, options.GraphicsLibrary)
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
	s := u.userInterfaceImpl.outsideSize.Load()
	return s.x, s.y
}

func (u *UserInterface) update() error {
	<-renderCh
	defer func() {
		renderEndCh <- struct{}{}
	}()

	w, h := u.outsideSize()
	s := theMonitor.DeviceScaleFactor()
	sw, sh := u.screenSize(w, h, s)
	if err := u.context.updateFrame(u.graphicsDriver, w, h, sw, sh, s, u, true); err != nil {
		return err
	}
	return nil
}

// screenSize returns the size of the rendering surface, in pixels.
//
// screenSize must be called on the same goroutine as update().
func (u *UserInterface) screenSize(outsideWidth, outsideHeight float64, deviceScaleFactor float64) (int, int) {
	if s := u.userInterfaceImpl.surfaceSize.Load(); s != nil {
		return s.x, s.y
	}
	// The platform reports no surface size, so derive it from the view's size in device-independent
	// pixels. That size is the view's pixel count divided by the scale factor, so rounding the
	// product lands back on the pixel count.
	return int(math.Round(outsideWidth * deviceScaleFactor)), int(math.Round(outsideHeight * deviceScaleFactor))
}

// SetSurfaceSize sets the size of the rendering surface, in pixels, as reported by the platform.
//
// SetSurfaceSize is concurrent safe.
func (u *UserInterface) SetSurfaceSize(width, height int) {
	u.userInterfaceImpl.surfaceSize.Store(&pointI{x: width, y: height})
}

// SetOutsideSize is called from mobile/ebitenmobileview.
//
// SetOutsideSize is concurrent safe.
func (u *UserInterface) SetOutsideSize(outsideWidth, outsideHeight float64) {
	u.userInterfaceImpl.outsideSize.Store(&pointF{x: outsideWidth, y: outsideHeight})
	u.refreshDisplayInfo()
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
	return u.foreground.Load()
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return false
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	// Do nothing
}

func (u *UserInterface) FPSMode() FPSModeType {
	return FPSModeType(u.fpsMode.Load())
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	u.fpsMode.Store(int32(mode))
	u.updateExplicitRenderingModeIfNeeded(mode)
}

func (u *UserInterface) updateExplicitRenderingModeIfNeeded(fpsMode FPSModeType) {
	r := u.currentRenderer()
	if r == nil {
		return
	}
	r.SetExplicitRenderingMode(fpsMode == FPSModeVsyncOffMinimum)
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.inputState.copyAndReset(inputState)
}

func (u *UserInterface) Window() Window {
	return &nullWindow{}
}

// displayInfoValues is the display info last recorded by refreshDisplayInfo,
// served to any thread by displayInfo.
type displayInfoValues struct {
	width  float64
	height float64
	scale  float64
}

var theDisplayInfo atomic.Pointer[displayInfoValues]

func (u *UserInterface) displayInfo() (int, int, float64, bool) {
	// Reading the display info here would require waiting for the main thread
	// on iOS, which can deadlock.
	v := theDisplayInfo.Load()
	if v == nil {
		return 0, 0, 1, false
	}
	width := int(math.Round(dipFromNativePixels(v.width, v.scale)))
	height := int(math.Round(dipFromNativePixels(v.height, v.scale)))
	return width, height, v.scale, true
}

type Monitor struct {
	cache atomic.Pointer[monitorCache]
}

// monitorCache is the monitor values and the tick they expire at. As there is no common way to
// detect monitor changes in Android and iOS, the values are invalidated regularly.
type monitorCache struct {
	monitor  monitor
	expireAt int64
}

type monitor struct {
	width             int
	height            int
	deviceScaleFactor float64
}

var theMonitor = &Monitor{}

func (m *Monitor) Name() string {
	return ""
}

func (m *Monitor) ensureValues() monitor {
	tick := theUI.Tick()
	if c := m.cache.Load(); c != nil && c.expireAt > tick {
		return c.monitor
	}

	width, height, scale, ok := theUI.displayInfo()
	if !ok {
		// This can happen e.g. when JVM is not ready.
		return monitor{
			width:             0,
			height:            0,
			deviceScaleFactor: 1,
		}
	}
	mon := monitor{
		width:             width,
		height:            height,
		deviceScaleFactor: scale,
	}
	m.cache.Store(&monitorCache{
		monitor:  mon,
		expireAt: tick + 1,
	})
	return mon
}

func (m *Monitor) DeviceScaleFactor() float64 {
	return m.ensureValues().deviceScaleFactor
}

func (m *Monitor) Size() (int, int) {
	mon := m.ensureValues()
	return mon.width, mon.height
}

func (u *UserInterface) AppendMonitors(mons []*Monitor) []*Monitor {
	return append(mons, theMonitor)
}

func (u *UserInterface) Monitor() *Monitor {
	return theMonitor
}

func (u *UserInterface) UpdateInput(keyPressedTimes, keyReleasedTimes [KeyMax + 1]InputTime, runes []rune, touches []TouchForInput, capsLock, numLock LockKeyState) {
	u.updateInputStateFromOutside(keyPressedTimes, keyReleasedTimes, runes, touches, capsLock, numLock)
	if FPSModeType(u.fpsMode.Load()) == FPSModeVsyncOffMinimum {
		// The renderer might not be set yet. In this case, the rendering request can be dropped
		// as the first rendering happens when the renderer is set.
		if r := u.currentRenderer(); r != nil {
			r.RequestRenderIfNeeded()
		}
	}
}

type Renderer interface {
	SetExplicitRenderingMode(explicitRendering bool)
	RequestRenderIfNeeded()
}

// rendererHolder holds a Renderer so that the renderer can be stored in an atomic pointer
// regardless of its concrete type.
type rendererHolder struct {
	renderer Renderer
}

func (u *UserInterface) SetRenderer(renderer Renderer) {
	u.renderer.Store(&rendererHolder{renderer: renderer})
	u.updateExplicitRenderingModeIfNeeded(FPSModeType(u.fpsMode.Load()))
}

func (u *UserInterface) currentRenderer() Renderer {
	h := u.renderer.Load()
	if h == nil {
		return nil
	}
	return h.renderer
}

func (u *UserInterface) ScheduleFrame() {
	if FPSModeType(u.fpsMode.Load()) != FPSModeVsyncOffMinimum {
		return
	}
	if r := u.currentRenderer(); r != nil {
		r.RequestRenderIfNeeded()
	}
}

func (u *UserInterface) updateIconIfNeeded() error {
	return nil
}

func IsScreenTransparentAvailable() bool {
	return false
}

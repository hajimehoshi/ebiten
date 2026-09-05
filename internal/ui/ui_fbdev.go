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

//go:build linux && !android && !nintendosdk && !playstation5

package ui

import (
	stdcontext "context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/fbdev"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

var _ uiBackend = (*fbdevBackend)(nil)

// fbdevBackend runs the game on a system that has no window system, presenting
// through the vendor EGL implementation of a framebuffer device.
type fbdevBackend struct {
	*UserInterface

	display    *fbdev.Display
	eglContext *fbdev.Context

	// frameCh wakes the game loop in FPSModeVsyncOffMinimum, where a frame runs
	// only when something asks for one.
	frameCh chan struct{}

	// monitor is the single monitor exposed to the game: the display itself.
	monitor *Monitor

	inputState InputState

	mu sync.Mutex
}

// maybeNewFbdevBackend returns a backend presenting on a framebuffer device, or
// the reason the device cannot be used.
func maybeNewFbdevBackend(u *UserInterface) (*fbdevBackend, error) {
	display, err := fbdev.OpenDisplay()
	if err != nil {
		return nil, err
	}

	b := &fbdevBackend{
		UserInterface: u,
		display:       display,
		frameCh:       make(chan struct{}, 1),
	}
	b.monitor = &Monitor{virtual: b}
	return b, nil
}

func (b *fbdevBackend) run(game Game, options *RunOptions) error {
	if options == nil {
		options = &RunOptions{}
	}

	b.mainThread = thread.NewOSThread()
	graphicscommand.SetOSThreadAsRenderThread()

	b.context = newContext(game, options.ScreenTransparent)

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the render thread.
	wg.Go(func() error {
		defer cancel()

		graphicscommand.LoopRenderThread(ctx)
		return nil
	})

	// Run the game thread.
	wg.Go(func() error {
		defer cancel()

		var initErr error
		b.mainThread.Call(func() {
			initErr = b.initOnMainThread(options)
		})
		if initErr != nil {
			return initErr
		}

		defer b.setRunningBackend(nil)

		return b.loopGame()
	})

	// Run the main thread. The loop is the thread's whole life, so a call arriving after
	// it ends is a no-op rather than a block forever.
	_ = b.mainThread.LoopAndStop(ctx)
	return wg.Wait()
}

func (b *fbdevBackend) initOnMainThread(options *RunOptions) error {
	c, err := fbdev.NewContext(b.display)
	if err != nil {
		return err
	}
	b.eglContext = c

	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	// The driver has nothing to present through until the EGL context is handed
	// over, so a driver that cannot take one is unusable here.
	p, ok := g.(interface{ SetPresenter(opengl.Presenter) })
	if !ok {
		return fmt.Errorf("ui: the graphics driver cannot present without a window system")
	}
	p.SetPresenter(c)

	b.graphicsDriver = g
	b.setGraphicsLibrary(lib)
	graphicscommand.SetVsyncEnabled(FPSModeType(b.fpsMode.Load()) == FPSModeVsyncOn)

	b.setRunningBackend(b)

	// Ask for the first frame. In FPSModeVsyncOffMinimum the game loop waits
	// for a request, and a framebuffer device raises no event that would stand
	// in for one, so nothing else would ever ask.
	b.ScheduleFrame()

	return nil
}

func (b *fbdevBackend) loopGame() (err error) {
	defer func() {
		graphicscommand.Terminate()
		b.mainThread.Call(func() {
			if b.eglContext != nil {
				err = errors.Join(err, b.eglContext.Close())
				b.eglContext = nil
			}
			b.setTerminated()
		})
	}()

	for {
		if err := b.updateGame(); err != nil {
			return err
		}
	}
}

func (b *fbdevBackend) updateGame() error {
	// In this mode a frame runs only when something asks for one, as there is
	// nothing to throttle the loop: the buffer swap does not block while vsync
	// is off. Only ScheduleFrame asks, as there is no window system to raise
	// events.
	if FPSModeType(b.fpsMode.Load()) == FPSModeVsyncOffMinimum {
		<-b.frameCh
	}

	if err := gamepad.Update(0, nil); err != nil {
		return err
	}

	w, h := b.outsideSize()
	sw, sh := b.screenSize()
	return b.context.updateFrame(b.graphicsDriver, w, h, sw, sh, b.deviceScaleFactor(), b.UserInterface, true)
}

// deviceScaleFactor implements virtualMonitorSource.
//
// A framebuffer device reports no physical size, so a logical pixel is a
// device pixel.
func (b *fbdevBackend) deviceScaleFactor() float64 {
	return 1
}

// outsideSize implements virtualMonitorSource.
//
// The surface covers the display, and nothing can resize it.
func (b *fbdevBackend) outsideSize() (width, height float64) {
	w, h := b.screenSize()
	return float64(w), float64(h)
}

// screenSize returns the size of the surface in pixels.
func (b *fbdevBackend) screenSize() (width, height int) {
	// The surface is created without a size, so the implementation chooses one.
	// Before it exists, the display's mode is the best answer available.
	if b.eglContext != nil {
		return b.eglContext.Size()
	}
	return b.display.Size()
}

func (b *fbdevBackend) readInputState(inputState *InputState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inputState.copyAndReset(inputState)
}

func (b *fbdevBackend) updateInputStateForFrame(deviceScaleFactor float64) error {
	return nil
}

func (b *fbdevBackend) KeyName(key Key) string {
	return ""
}

func (b *fbdevBackend) updateIconIfNeeded() error {
	return nil
}

func (b *fbdevBackend) IsFocused() bool {
	return true
}

func (b *fbdevBackend) IsFullscreen() bool {
	// The surface always covers the display, which is not the same as the
	// fullscreen a window system offers.
	return false
}

func (b *fbdevBackend) SetFullscreen(fullscreen bool) {
}

func (b *fbdevBackend) CursorMode() CursorMode {
	return CursorModeHidden
}

func (b *fbdevBackend) SetCursorMode(mode CursorMode) {
}

func (b *fbdevBackend) applyCursorShape() {
}

func (b *fbdevBackend) applyFPSMode() {
	b.RunOnMainThread(func() {
		graphicscommand.SetVsyncEnabled(FPSModeType(b.fpsMode.Load()) == FPSModeVsyncOn)
	})
}

func (b *fbdevBackend) ScheduleFrame() {
	// The game loop can be waiting for this, so never block on a wakeup that is
	// already pending.
	select {
	case b.frameCh <- struct{}{}:
	default:
	}
}

func (b *fbdevBackend) Window() backendWindow {
	return &nullWindow{}
}

func (b *fbdevBackend) Monitor() *Monitor {
	return b.monitor
}

func (b *fbdevBackend) appendMonitors(monitors []*Monitor) []*Monitor {
	return append(monitors, b.monitor)
}

func (b *fbdevBackend) RunOnMainThread(f func()) {
	b.mainThread.Call(f)
}

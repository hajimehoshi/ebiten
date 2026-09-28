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

var _ uiBackend = (*consoleBackend)(nil)

type consoleContext interface {
	opengl.Presenter
	Size() (int, int)
	Close() error
}

// consoleBackend runs the game without a window system, using either GBM/KMS
// or a framebuffer device for presentation.
type consoleBackend struct {
	*UserInterface

	eglContext    consoleContext
	newContext    func() (consoleContext, error)
	closeDisplay  func() error
	sizeMu        sync.RWMutex
	width, height int

	// frameCh wakes the game loop in FPSModeVsyncOffMinimum, where a frame runs
	// only when something asks for one.
	frameCh chan struct{}

	// monitor is the single monitor exposed to the game: the display itself.
	monitor *Monitor

	inputState InputState

	mu sync.Mutex
}

func newConsoleBackend(u *UserInterface, width, height int, c consoleContext, newContext func() (consoleContext, error), closeDisplay func() error) *consoleBackend {
	b := &consoleBackend{
		UserInterface: u,
		eglContext:    c,
		newContext:    newContext,
		closeDisplay:  closeDisplay,
		width:         width,
		height:        height,
		frameCh:       make(chan struct{}, 1),
	}
	b.monitor = &Monitor{virtual: b}
	return b
}

// maybeNewFbdevBackend returns a backend presenting on a framebuffer device, or
// the reason the device cannot be used.
func maybeNewFbdevBackend(u *UserInterface) (uiBackend, error) {
	display, err := fbdev.OpenDisplay()
	if err != nil {
		return nil, err
	}

	width, height := display.Size()
	return newConsoleBackend(u, width, height, nil, func() (consoleContext, error) {
		return fbdev.NewContext(display)
	}, nil), nil
}

func (b *consoleBackend) run(game Game, options *RunOptions) error {
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

		type args struct {
			b       *consoleBackend
			options *RunOptions
		}
		if err := thread.CallWithArgAndResult(b.mainThread, func(a args) error {
			return a.b.initOnMainThread(a.options)
		}, args{b: b, options: options}); err != nil {
			return err
		}

		defer b.setRunningBackend(nil)

		return b.loopGame()
	})

	// Run the main thread. The loop is the thread's whole life, so a call arriving after
	// it ends is a no-op rather than a block forever.
	_ = b.mainThread.LoopAndStop(ctx)
	return wg.Wait()
}

func (b *consoleBackend) initOnMainThread(options *RunOptions) (err error) {
	if b.eglContext == nil {
		c, err := b.newContext()
		if err != nil {
			return err
		}
		b.eglContext = c
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, b.closeOnMainThread())
		}
	}()
	c := b.eglContext
	width, height := c.Size()
	b.sizeMu.Lock()
	b.width, b.height = width, height
	b.sizeMu.Unlock()

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

func (b *consoleBackend) loopGame() (err error) {
	defer func() {
		graphicscommand.Terminate()
		closeErr := thread.CallWithArgAndResult(b.mainThread, func(b *consoleBackend) error {
			defer b.setTerminated()
			return b.closeOnMainThread()
		}, b)
		err = errors.Join(err, closeErr)
	}()

	for {
		if err := b.updateGame(); err != nil {
			return err
		}
	}
}

func (b *consoleBackend) closeOnMainThread() error {
	var err error
	if b.eglContext != nil {
		err = errors.Join(err, b.eglContext.Close())
		b.eglContext = nil
	}
	if b.closeDisplay != nil {
		err = errors.Join(err, b.closeDisplay())
		b.closeDisplay = nil
	}
	return err
}

func (b *consoleBackend) updateGame() error {
	// In this mode a frame runs only when something asks for one, as there is
	// only ScheduleFrame asks for a frame: there is no window system to raise
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
// Console displays report no physical size, so a logical pixel is a device
// pixel.
func (b *consoleBackend) deviceScaleFactor() float64 {
	return 1
}

// outsideSize implements virtualMonitorSource.
//
// The surface covers the display, and nothing can resize it.
func (b *consoleBackend) outsideSize() (width, height float64) {
	w, h := b.screenSize()
	return float64(w), float64(h)
}

// screenSize returns the size of the surface in pixels.
func (b *consoleBackend) screenSize() (width, height int) {
	b.sizeMu.RLock()
	defer b.sizeMu.RUnlock()
	return b.width, b.height
}

func (b *consoleBackend) readInputState(inputState *InputState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inputState.copyAndReset(inputState)
}

func (b *consoleBackend) updateInputStateForFrame(deviceScaleFactor float64) error {
	return nil
}

func (b *consoleBackend) KeyName(key Key) string {
	return ""
}

func (b *consoleBackend) updateIconIfNeeded() error {
	return nil
}

func (b *consoleBackend) IsFocused() bool {
	return true
}

func (b *consoleBackend) IsFullscreen() bool {
	// The surface always covers the display, which is not the same as the
	// fullscreen a window system offers.
	return false
}

func (b *consoleBackend) SetFullscreen(fullscreen bool) {
}

func (b *consoleBackend) CursorMode() CursorMode {
	return CursorModeHidden
}

func (b *consoleBackend) SetCursorMode(mode CursorMode) {
}

func (b *consoleBackend) applyCursorShape() {
}

func (b *consoleBackend) applyFPSMode() {
	b.RunOnMainThread(func() {
		graphicscommand.SetVsyncEnabled(FPSModeType(b.fpsMode.Load()) == FPSModeVsyncOn)
	})
}

func (b *consoleBackend) ScheduleFrame() {
	// The game loop can be waiting for this, so never block on a wakeup that is
	// already pending.
	select {
	case b.frameCh <- struct{}{}:
	default:
	}
}

func (b *consoleBackend) Window() backendWindow {
	return &nullWindow{}
}

func (b *consoleBackend) Monitor() *Monitor {
	return b.monitor
}

func (b *consoleBackend) appendMonitors(monitors []*Monitor) []*Monitor {
	return append(monitors, b.monitor)
}

func (b *consoleBackend) RunOnMainThread(f func()) {
	thread.Call(b.mainThread, f)
}

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

package ui

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

func (u *UserInterface) initializePlatform() error {
	return nil
}

func (u *UserInterface) setApplePressAndHoldEnabled(enabled bool) {
	// Do nothings.
}

type graphicsDriverCreatorImpl struct {
	transparent bool
	colorSpace  graphicsdriver.ColorSpace
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	var dxErr error
	var glErr error
	if winver.IsWindows10OrGreater() {
		d, err := g.newDirectX()
		if err == nil {
			return d, GraphicsLibraryDirectX, nil
		}
		dxErr = err

		o, err := g.newOpenGL()
		if err == nil {
			return o, GraphicsLibraryOpenGL, nil
		}
		glErr = err
	} else {
		// Creating a swap chain on an older machine than Windows 10 might fail (#2613).
		// Prefer OpenGL to DirectX.
		o, err := g.newOpenGL()
		if err == nil {
			return o, GraphicsLibraryOpenGL, nil
		}
		glErr = err

		// Initializing OpenGL can fail, though this is pretty rare.
		d, err := g.newDirectX()
		if err == nil {
			return d, GraphicsLibraryDirectX, nil
		}
		dxErr = err
	}

	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: DirectX: %v, OpenGL: %v", dxErr, glErr)
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (g *graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	if g.transparent {
		return nil, errors.New("ui: DirectX is not available with a transparent window")
	}
	return directx.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: Metal is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
}

// glfwMonitorSizeInGLFWPixels must be called from the main thread.
func glfwMonitorSizeInGLFWPixels(m *glfw.Monitor) (int, int, error) {
	vm, err := m.GetVideoMode()
	if err != nil {
		return 0, 0, err
	}
	return vm.Width, vm.Height, nil
}

func dipFromGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x / deviceScaleFactor
}

func dipToGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x * deviceScaleFactor
}

func (u *UserInterface) adjustWindowPosition(x, y int, monitor *Monitor) (int, int, error) {
	if microsoftgdk.IsXbox() {
		return x, y, nil
	}

	// If a window is not decorated, the window should be able to reach the top of the screen (#3118).
	d, err := u.window.GetAttrib(glfw.Decorated)
	if err != nil {
		return 0, 0, err
	}
	if d == glfw.False {
		return x, y, nil
	}

	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	// As the video width/height might be wrong,
	// adjust x/y at least to enable to handle the window (#328)
	if x < mx {
		x = mx
	}
	t, err := _GetSystemMetrics(_SM_CYCAPTION)
	if err != nil {
		return 0, 0, err
	}
	if y < my+int(t) {
		y = my + int(t)
	}
	return x, y, nil
}

func initialMonitorByOS() (*Monitor, error) {
	if microsoftgdk.IsXbox() {
		return theMonitors.primaryMonitor(), nil
	}

	px, py, err := _GetCursorPos()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, nil
		}
		return nil, err
	}
	x, y := int(px), int(py)

	// Find the monitor including the cursor.
	return theMonitors.monitorFromPosition(x, y), nil
}

func monitorFromWindowByOS(w *glfw.Window) (*Monitor, error) {
	if microsoftgdk.IsXbox() {
		return theMonitors.primaryMonitor(), nil
	}
	window, err := w.GetWin32Window()
	if err != nil {
		return nil, err
	}
	return monitorFromWin32Window(window), nil
}

func monitorFromWin32Window(w windows.HWND) *Monitor {
	// Get the current monitor by the window handle instead of the window position. It is because the window
	// position is not reliable in some cases e.g. when the window is put across multiple monitors.

	m := _MonitorFromWindow(w, _MONITOR_DEFAULTTONEAREST)
	if m == 0 {
		// monitorFromWindow can return error on Wine. Ignore this.
		return nil
	}

	mi, err := _GetMonitorInfoW(m)
	if err != nil {
		panic(err)
	}

	x, y := int(mi.rcMonitor.left), int(mi.rcMonitor.top)
	for _, m := range theMonitors.append(nil) {
		mx := m.boundsInGLFWPixels.Min.X
		my := m.boundsInGLFWPixels.Min.Y
		if mx == x && my == y {
			return m
		}
	}
	return nil
}

func (u *UserInterface) nativeWindow() (uintptr, error) {
	w, err := u.window.GetWin32Window()
	return uintptr(w), err
}

func (u *UserInterface) isNativeFullscreen() (bool, error) {
	return false, nil
}

func (u *UserInterface) isNativeFullscreenAvailable() bool {
	return false
}

func (u *UserInterface) setNativeFullscreen(fullscreen bool) error {
	panic(fmt.Sprintf("ui: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *UserInterface) adjustViewSizeAfterFullscreen() error {
	return nil
}

func (u *UserInterface) setWindowResizingModeForOS(mode WindowResizingMode) error {
	return nil
}

func initializeWindowAfterCreation(w *glfw.Window) error {
	return nil
}

func (u *UserInterface) skipTaskbar() error {
	// S_FALSE is returned when CoInitializeEx is nested. This is a successful case.
	if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil && !errors.Is(err, syscall.Errno(windows.S_FALSE)) {
		return err
	}
	// CoUninitialize should be called even when CoInitializeEx returns S_FALSE.
	defer windows.CoUninitialize()

	ptr, err := _CoCreateInstance(&_CLSID_TaskbarList, nil, _CLSCTX_SERVER, &_IID_ITaskbarList)
	if err != nil {
		return err
	}

	t := (*_ITaskbarList)(ptr)
	defer t.Release()

	w, err := u.window.GetWin32Window()
	if err != nil {
		return err
	}
	if err := t.DeleteTab(w); err != nil {
		return err
	}

	return nil
}

func (u *UserInterface) setDocumentEdited(edited bool) error {
	return nil
}

func (u *UserInterface) afterWindowCreation() error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	// By default, IME should be disabled (#2918).
	w, err := u.window.GetWin32Window()
	if err != nil {
		return err
	}
	c, err := _ImmAssociateContext(w, 0)
	if err != nil {
		return err
	}
	u.immContext = c
	return nil
}

// RestoreIMMContextOnMainThread is called from the main thread.
// The textinput package invokes RestoreIMMContextOnMainThread to enable IME inputting.
func (u *UserInterface) RestoreIMMContextOnMainThread() error {
	w, err := u.window.GetWin32Window()
	if err != nil {
		return err
	}
	if _, err := _ImmAssociateContext(w, u.immContext); err != nil {
		return err
	}
	u.immContext = 0
	return nil
}

func init() {
	if microsoftgdk.IsXbox() {
		// TimeBeginPeriod might not be defined in Xbox.
		return
	}
	// Use a better timer resolution (golang/go#44343).
	// An error is ignored. The application is still valid even if a higher resolution timer is not available.
	// TODO: This might not be necessary from Go 1.23.
	_ = windows.TimeBeginPeriod(1)
}

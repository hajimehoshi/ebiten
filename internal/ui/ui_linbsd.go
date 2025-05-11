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

//go:build (freebsd || (linux && !android) || netbsd || openbsd) && !nintendosdk && !playstation5

package ui

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
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

// glfwMonitorSizeInGLFWPixels must be called from the main thread.
func glfwMonitorSizeInGLFWPixels(m *glfw.Monitor) (int, int, error) {
	vm, err := m.GetVideoMode()
	if err != nil {
		return 0, 0, err
	}
	physWidth, physHeight := vm.Width, vm.Height

	// TODO: if glfw/glfw#1961 gets fixed, this function may need revising.
	// In case GLFW decides to switch to returning logical pixels, we can just return 1.

	// Note: GLFW currently returns physical pixel sizes,
	// but we need to predict the window system-side size of the fullscreen window
	// for Ebitengine's `(*Monitor).Size()` public API.
	// Also at the moment we need this prior to switching to fullscreen, but that might be replaceable.
	// So this function computes the ratio of physical per logical pixels.
	xconn, err := xgb.NewConn()
	if err != nil {
		// No X11 connection?
		// Assume we're on pure Wayland then.
		// GLFW/Wayland shouldn't be having this issue.
		return physWidth, physHeight, nil
	}
	defer xconn.Close()

	if err := randr.Init(xconn); err != nil {
		// No RANDR extension? No problem.
		return physWidth, physHeight, nil
	}

	root := xproto.Setup(xconn).DefaultScreen(xconn).Root
	res, err := randr.GetScreenResourcesCurrent(xconn, root).Reply()
	if err != nil {
		// Likely means RANDR is not working. No problem.
		return physWidth, physHeight, nil
	}

	monitorX, monitorY, err := m.GetPos()
	if err != nil {
		// TODO: Is it OK to ignore this error?
		return physWidth, physHeight, nil
	}

	for _, crtc := range res.Crtcs[:res.NumCrtcs] {
		info, err := randr.GetCrtcInfo(xconn, crtc, res.ConfigTimestamp).Reply()
		if err != nil {
			// This Crtc is bad. Maybe just got disconnected?
			continue
		}
		if info.NumOutputs == 0 {
			// This Crtc is not connected to any output.
			// In other words, a disabled monitor.
			continue
		}
		if int(info.X) == monitorX && int(info.Y) == monitorY {
			return int(info.Width), int(info.Height), nil
		}
	}

	// Monitor not known to XRandR. Weird.
	return physWidth, physHeight, nil
}

func dipFromGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x / deviceScaleFactor
}

func dipToGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x * deviceScaleFactor
}

func (u *UserInterface) adjustWindowPosition(x, y int, monitor *Monitor) (int, int, error) {
	return x, y, nil
}

func initialMonitorByOS() (*Monitor, error) {
	xconn, err := xgb.NewConn()
	if err != nil {
		// Assume we're on pure Wayland then.
		return nil, nil
	}
	defer xconn.Close()

	root := xproto.Setup(xconn).DefaultScreen(xconn).Root
	rep, err := xproto.QueryPointer(xconn, root).Reply()
	if err != nil {
		return nil, err
	}
	x, y := int(rep.RootX), int(rep.RootY)

	// Find the monitor including the cursor.
	return theMonitors.monitorFromPosition(x, y), nil
}

func monitorFromWindowByOS(_ *glfw.Window) (*Monitor, error) {
	// TODO: Implement this correctly. (#1119).
	return nil, nil
}

func (u *UserInterface) nativeWindow() (uintptr, error) {
	// TODO: Implement this.
	return 0, nil
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
	// Show the window once before getting the position of the window.
	// On Linux/Unix, the window position is not reliable before showing.
	if err := w.Show(); err != nil {
		return err
	}

	// Hiding the window makes the position unreliable again. Do not call w.Hide() here (#1829)
	// Calling Hide is problematic especially on XWayland and/or Sway.
	// Apparently the window state is inconsistent just after the window is created, but we are not sure.
	// For more details, see the discussion in #1829.
	return nil
}

func (u *UserInterface) skipTaskbar() error {
	return nil
}

func (u *UserInterface) setDocumentEdited(edited bool) error {
	return nil
}

func (u *UserInterface) afterWindowCreation() error {
	return nil
}

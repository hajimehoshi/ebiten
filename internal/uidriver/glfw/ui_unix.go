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

//go:build (dragonfly || freebsd || linux || netbsd || openbsd || solaris) && !android
// +build dragonfly freebsd linux netbsd openbsd solaris
// +build !android

package glfw

import (
	"fmt"
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"
)

type videoModeScaleCacheKey struct{ X, Y int }

var videoModeScaleCache = map[videoModeScaleCacheKey]float64{}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {
	for k := range videoModeScaleCache {
		delete(videoModeScaleCache, k)
	}
}

// videoModeScale must be called from the main thread.
func videoModeScale(m *glfw.Monitor) float64 {
	// Caching wrapper for videoModeScaleUncached as
	// videoModeScaleUncached may be expensive (uses blocking calls on X connection)
	// and public ScreenSizeInFullscreen API needs the videoModeScale.
	monitorX, monitorY := m.GetPos()
	cacheKey := videoModeScaleCacheKey{X: monitorX, Y: monitorY}
	if cached, ok := videoModeScaleCache[cacheKey]; ok {
		return cached
	}

	scale := videoModeScaleUncached(m)
	videoModeScaleCache[cacheKey] = scale
	return scale
}

// videoModeScaleUncached must be called from the main thread.
func videoModeScaleUncached(m *glfw.Monitor) float64 {
	// TODO: if glfw/glfw#1961 gets fixed, this function may need revising.
	// In case GLFW decides to switch to returning logical pixels, we can just return 1.

	// Note: GLFW currently returns physical pixel sizes,
	// but we need to predict the window system-side size of the fullscreen window
	// for Ebiten's `ScreenSizeInFullscreen` public API.
	// Also at the moment we need this prior to switching to fullscreen, but that might be replacable.
	// So this function computes the ratio of physical per logical pixels.
	xconn, err := xgb.NewConn()
	if err != nil {
		// No X11 connection?
		// Assume we're on pure Wayland then.
		// GLFW/Wayland shouldn't be having this issue.
		return 1
	}
	defer xconn.Close()

	if err := randr.Init(xconn); err != nil {
		// No RANDR extension? No problem.
		return 1
	}

	root := xproto.Setup(xconn).DefaultScreen(xconn).Root
	res, err := randr.GetScreenResourcesCurrent(xconn, root).Reply()
	if err != nil {
		// Likely means RANDR is not working. No problem.
		return 1
	}

	monitorX, monitorY := m.GetPos()

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
			xWidth, xHeight := info.Width, info.Height
			vm := m.GetVideoMode()
			physWidth, physHeight := vm.Width, vm.Height
			// Return one scale, even though there may be separate X and Y scales.
			// Return the _larger_ scale, as this would yield a letterboxed display on mismatch, rather than a cut-off one.
			scale := math.Max(float64(physWidth)/float64(xWidth), float64(physHeight)/float64(xHeight))
			return scale
		}
	}

	// Monitor not known to XRandR. Weird.
	return 1
}

// fromGLFWMonitorPixel must be called from the main thread.
func (u *UserInterface) fromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / (videoModeScale(monitor) * u.deviceScaleFactor(monitor))
}

// fromGLFWPixel must be called from the main thread.
func (u *UserInterface) fromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / u.deviceScaleFactor(monitor)
}

// toGLFWPixel must be called from the main thread.
func (u *UserInterface) toGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x * u.deviceScaleFactor(monitor)
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func initialMonitorByOS() *glfw.Monitor {
	return nil
}

func currentMonitorByOS(_ *glfw.Window) *glfw.Monitor {
	// TODO: Implement this correctly. (#1119).
	return nil
}

func (u *UserInterface) nativeWindow() uintptr {
	// TODO: Implement this.
	return 0
}

func (u *UserInterface) isNativeFullscreen() bool {
	return false
}

func (u *UserInterface) setNativeCursor(shape driver.CursorShape) {
	// TODO: Use native API in the future (#1571)
	u.window.SetCursor(glfwSystemCursors[shape])
}

func (u *UserInterface) isNativeFullscreenAvailable() bool {
	return false
}

func (u *UserInterface) setNativeFullscreen(fullscreen bool) {
	panic(fmt.Sprintf("glfw: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *UserInterface) adjustViewSize() {
}

func initializeWindowAfterCreation(w *glfw.Window) {
	// Show the window once before getting the position of the window.
	// On Linux/Unix, the window position is not reliable before showing.
	w.Show()

	// Hiding the window makes the position unreliable again. Do not call w.Hide() here (#1829)
	// Calling Hide is problematic especially on XWayland and/or Sway.
	// Apparently the window state is inconsistent just after the window is created, but we are not sure.
	// For more details, see the discussion in #1829.
}

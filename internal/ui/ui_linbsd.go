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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

func (u *UserInterface) initializePlatform() error {
	return nil
}

func (u *glfwBackend) setApplePressAndHoldEnabled(enabled bool) {
	// Do nothing.
}

type graphicsDriverCreatorImpl struct {
	transparent bool
	colorSpace  color.ColorSpace
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

func isGLXExtensionForGL2Available() bool {
	out, err := exec.Command("glxinfo").Output()
	if err != nil {
		return false
	}

	const (
		indent = "    "
		ext    = "GLX_EXT_create_context_es2_profile"
	)

	var listingExtensions bool
	for line := range strings.Lines(string(out)) {
		line = strings.TrimRight(line, "\r\n")
		if !listingExtensions {
			if line == "GLX extensions:" {
				listingExtensions = true
			}
			continue
		}

		if !strings.HasPrefix(line, indent) {
			listingExtensions = false
			break
		}

		for len(line) > 0 {
			head, tail, _ := strings.Cut(line, ",")
			if strings.TrimSpace(head) == ext {
				return true
			}
			line = tail
		}
	}
	return false
}

// setOpenGLWindowHints sets the GLFW hints to create a window with an OpenGL context.
//
// setOpenGLWindowHints must be called from the main thread.
func (u *glfwBackend) setOpenGLWindowHints() error {
	var isES bool
	if g, ok := u.graphicsDriver.(interface{ IsES() bool }); ok {
		isES = g.IsES()
	}

	if isES {
		if err := glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.ContextVersionMajor, 3); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.ContextVersionMinor, 0); err != nil {
			return err
		}
		// Use GLX if the extension allows, or use EGL otherwise.
		// Prefer GLX since EGL might not work well on Wayland (#3152).
		if !isGLXExtensionForGL2Available() {
			if err := glfw.WindowHint(glfw.ContextCreationAPI, glfw.EGLContextAPI); err != nil {
				return err
			}
		}
		return nil
	}

	if err := glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI); err != nil {
		return err
	}
	if err := glfw.WindowHint(glfw.ContextVersionMajor, 3); err != nil {
		return err
	}
	if err := glfw.WindowHint(glfw.ContextVersionMinor, 2); err != nil {
		return err
	}
	return nil
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
	if !ensureX11() {
		// No X11 connection? Assume we're on pure Wayland then.
		// GLFW/Wayland shouldn't be having this issue.
		return physWidth, physHeight, nil
	}

	display, err := glfw.GetX11Display()
	if err != nil || display == 0 {
		return physWidth, physHeight, nil
	}

	crtc, err := m.GetX11Adapter()
	if err != nil || crtc == 0 {
		return physWidth, physHeight, nil
	}

	w, h, ok := x11CrtcSize(display, uint(crtc))
	if !ok {
		// Monitor not known to XRandR. Weird.
		return physWidth, physHeight, nil
	}
	return w, h, nil
}

func dipFromGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x / deviceScaleFactor
}

func dipToGLFWPixel(x float64, deviceScaleFactor float64) float64 {
	return x * deviceScaleFactor
}

func (u *glfwBackend) adjustWindowPosition(x, y int, monitor *Monitor) (int, int, error) {
	return x, y, nil
}

func initialMonitorByOS() (*Monitor, error) {
	if !ensureX11() {
		// Assume we're on pure Wayland then.
		return nil, nil
	}

	display, err := glfw.GetX11Display()
	if err != nil || display == 0 {
		return nil, nil
	}

	x, y, _, ok := x11QueryPointer(display)
	if !ok {
		return nil, nil
	}

	// Find the monitor including the cursor.
	return theMonitors.monitorFromPosition(x, y), nil
}

func monitorFromWindowByOS(_ *glfw.Window) (*Monitor, error) {
	// TODO: Implement this correctly. (#1119).
	return nil, nil
}

func (u *glfwBackend) nativeWindow() (uintptr, error) {
	// TODO: Implement this.
	return 0, nil
}

func (u *glfwBackend) isWindowOccluded() (bool, error) {
	// The tick rate doesn't drop for an occluded window on X11, so skipping the present is not
	// needed to keep the game running. Occlusion is reported only as VisibilityNotify events,
	// which a compositing manager suppresses by redirecting windows to offscreen storage.
	return false, errors.ErrUnsupported
}

func (u *glfwBackend) isNativeFullscreen() (bool, error) {
	return false, nil
}

func (u *glfwBackend) isNativeFullscreenAvailable() bool {
	return false
}

func (u *glfwBackend) setNativeFullscreen(fullscreen bool) error {
	panic(fmt.Sprintf("ui: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *glfwBackend) setWindowResizingModeForOS(mode WindowResizingMode) error {
	return nil
}

func (u *glfwBackend) initializeWindowAfterCreation(w *glfw.Window) error {
	// A window that must stay invisible is never shown (#3495).
	// Its position stays unreliable, which doesn't matter as the window is not on the screen.
	if !u.desktopWindow.isInitWindowVisible() {
		return nil
	}

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

func (u *glfwBackend) skipTaskbar() error {
	return nil
}

func (u *glfwBackend) setDocumentEdited(edited bool) error {
	return nil
}

func (u *glfwBackend) afterWindowCreation() error {
	return nil
}

// setWindowColorModeImpl must be called from the main thread.
func (u *glfwBackend) setWindowColorModeImpl(mode colormode.ColorMode) error {
	if !ensureX11() {
		// Assume we're on pure Wayland then.
		return nil
	}

	display, err := glfw.GetX11Display()
	if err != nil {
		return err
	}
	if display == 0 {
		return nil
	}

	// Get the X11 window ID from GLFW
	window, err := u.window.GetX11Window()
	if err != nil {
		return err
	}

	var themeVariant string
	switch mode {
	case colormode.Light:
		themeVariant = "light"
	case colormode.Dark:
		themeVariant = "dark"
	case colormode.Unknown:
		// Keep themeVariant empty.
	}

	x11SetWindowThemeVariant(display, window, themeVariant)
	return nil
}

func (u *glfwBackend) syncModKeysFromOS() {}

// syncLockKeysFromOS updates the lock key state to the current OS state.
// Must be called on the main thread.
func (u *glfwBackend) syncLockKeysFromOS() {
	if !ensureX11() {
		return
	}
	display, err := glfw.GetX11Display()
	if err != nil || display == 0 {
		return
	}
	_, _, mask, ok := x11QueryPointer(display)
	if !ok {
		return
	}

	caps := NewLockKeyStateFromBool(mask&xLockMask != 0)
	num := NewLockKeyStateFromBool(mask&xMod2Mask != 0)
	u.input.setLockKeys(caps, num)
}

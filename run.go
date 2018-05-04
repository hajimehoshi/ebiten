// Copyright 2014 Hajime Hoshi
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

package ebiten

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/png"
	"github.com/hajimehoshi/ebiten/internal/shareable"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// FPS represents how many times game updating happens in a second (60).
const FPS = clock.FPS

// CurrentFPS returns the current number of frames per second of rendering.
//
// The returned value represents how many times rendering happens in a second and
// NOT how many times logical game updating (a passed function to Run) happens.
// Note that logical game updating is assured to happen 60 times in a second.
//
// This function is concurrent-safe.
func CurrentFPS() float64 {
	return clock.CurrentFPS()
}

var (
	isRunningSlowly = int32(0)
)

func setRunningSlowly(slow bool) {
	v := int32(0)
	if slow {
		v = 1
	}
	atomic.StoreInt32(&isRunningSlowly, v)
}

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS of rendering.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing screen,
// when IsRunningSlowly is true.
//
// The typical code with IsRunningSlowly is this:
//
//    func update(screen *ebiten.Image) error {
//
//        // Update the state.
//
//        // When IsRunningSlowly is true, the rendered result is not adopted.
//        // Skip rendering then.
//        if ebiten.IsRunningSlowly() {
//            return nil
//        }
//
//        // Draw something to the screen.
//
//        return nil
//    }
//
// This function is concurrent-safe.
func IsRunningSlowly() bool {
	return atomic.LoadInt32(&isRunningSlowly) != 0
}

var theGraphicsContext atomic.Value

func run(width, height int, scale float64, title string, g *graphicsContext, mainloop bool) error {
	if err := ui.Run(width, height, scale, title, g, mainloop); err != nil {
		if err == ui.RegularTermination {
			return nil
		}
		return err
	}
	return nil
}

type imageDumper struct {
	f func(screen *Image) error

	keyState map[Key]int

	hasScreenshotKey bool
	screenshotKey    Key
	toTakeScreenshot bool

	hasDumpInternalImagesKey bool
	dumpInternalImagesKey    Key
	toDumpInternalImages     bool
}

func (i *imageDumper) update(screen *Image) error {
	if err := i.f(screen); err != nil {
		return err
	}

	// If keyState is nil, all values are not initialized.
	if i.keyState == nil {
		i.keyState = map[Key]int{}

		if keyname := os.Getenv("EBITEN_SCREENSHOT_KEY"); keyname != "" {
			if key, ok := keyNameToKey(keyname); ok {
				i.hasScreenshotKey = true
				i.screenshotKey = key
			}
		}
		if keyname := os.Getenv("EBITEN_INTERNAL_IMAGES_KEY"); keyname != "" {
			if key, ok := keyNameToKey(keyname); ok {
				i.hasDumpInternalImagesKey = true
				i.dumpInternalImagesKey = key
			}
		}
	}

	keys := map[Key]struct{}{}
	if i.hasScreenshotKey {
		keys[i.screenshotKey] = struct{}{}
	}
	if i.hasDumpInternalImagesKey {
		keys[i.dumpInternalImagesKey] = struct{}{}
	}

	for key := range keys {
		if IsKeyPressed(key) {
			i.keyState[key]++
			if i.keyState[key] == 1 {
				if i.hasScreenshotKey && key == i.screenshotKey {
					i.toTakeScreenshot = true
				}
				if i.hasDumpInternalImagesKey && key == i.dumpInternalImagesKey {
					i.toDumpInternalImages = true
				}
			}
		} else {
			i.keyState[key] = 0
		}
	}

	if IsRunningSlowly() {
		return nil
	}

	if i.toTakeScreenshot {
		dump := func() (string, error) {
			f, err := ioutil.TempFile("", "ebiten_screenshot_")
			if err != nil {
				return "", err
			}
			defer f.Close()

			if err := png.Encode(f, screen); err != nil {
				return "", err
			}
			return f.Name(), nil
		}

		name, err := dump()
		if err != nil {
			return err
		}
		if err := os.Rename(name, name+".png"); err != nil {
			return err
		}

		i.toTakeScreenshot = false
		fmt.Fprintf(os.Stderr, "Saved screenshot: %s.png\n", name)
	}

	if i.toDumpInternalImages {
		dir, err := ioutil.TempDir("", "ebiten_internal_images_")
		if err != nil {
			return err
		}

		dump := func(img image.Image, index int) error {
			filename := filepath.Join(dir, fmt.Sprintf("%d.png", index))
			f, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := png.Encode(f, img); err != nil {
				return err
			}
			return nil
		}

		images, err := shareable.Images()
		if err != nil {
			return err
		}
		for i, img := range images {
			if err := dump(img, i); err != nil {
				return err
			}
		}

		i.toDumpInternalImages = false

		fmt.Fprintf(os.Stderr, "Dumped the internal images at: %s\n", dir)
	}

	return nil
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
// The screen size is based on the given values (width and height).
//
// A window size is based on the given values (width, height and scale).
// scale is used to enlarge the screen.
// Note that the actual screen is multiplied not only by the given scale but also
// by the device scale on high-DPI display.
// If you pass inverse of the device scale,
// you can disable this automatical device scaling as a result.
// You can get the device scale by DeviceScaleFactor function.
//
// Run must be called from the OS main thread.
// Note that Ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// The given function f is guaranteed to be called 60 times a second
// even if a rendering frame is skipped.
// f is not called when the window is in background by default.
// This setting is configurable with SetRunnableInBackground.
//
// The given scale is ignored on fullscreen mode or gomobile-build mode.
//
// Run returns error when 1) OpenGL error happens, 2) audio error happens or 3) f returns error.
// In the case of 3), Run returns the same error.
//
// The size unit is device-independent pixel.
//
// Don't call Run twice or more in one process.
func Run(f func(*Image) error, width, height int, scale float64, title string) error {
	f = (&imageDumper{f: f}).update

	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g, true); err != nil {
			ch <- err
			return
		}
	}()
	// TODO: Use context in Go 1.7?
	if err := ui.RunMainThreadLoop(ch); err != nil {
		return err
	}
	return nil
}

// RunWithoutMainLoop runs the game, but don't call the loop on the main (UI) thread.
// Different from Run, this function returns immediately.
//
// Ebiten users should NOT call this function.
// Instead, functions in github.com/hajimehoshi/ebiten/mobile package calls this.
func RunWithoutMainLoop(f func(*Image) error, width, height int, scale float64, title string) <-chan error {
	f = (&imageDumper{f: f}).update

	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g, false); err != nil {
			ch <- err
			return
		}
	}()
	return ch
}

// MonitorSize returns the monitor size in device-independent pixels.
//
// On browsers, MonitorSize returns the 'window' size, not 'screen' size since an Ebiten game
// should not know the outside of the window object.
// For more detials, see SetFullscreen API comment.
//
// On mobiles, MonitorSize returns (0, 0) so far.
//
// Note that MonitorSize returns the 'primary' monitor size, which is the monitor
// where taskbar is present (Windows) or menubar is present (macOS).
//
// If you use this for screen size with SetFullscreen(true), you can get the fullscreen mode
// which size is well adjusted with the monitor.
//
//     w, h := MonitorSize()
//     ebiten.SetFullscreen(true)
//     ebiten.Run(update, w, h, 1, "title")
//
// Furthermore, you can use them with DeviceScaleFactor(), you can get the finest
// fullscreen mode.
//
//     s := ebiten.DeviceScaleFactor()
//     w, h := MonitorSize()
//     ebiten.SetFullscreen(true)
//     ebiten.Run(update, int(float64(w) * s), int(float64(h) * s), 1/s, "title")
//
// For actual example, see examples/fullscreen
//
// MonitorSize is concurrent-safe.
func MonitorSize() (int, int) {
	return ui.MonitorSize()
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
//
// Unit is device-independent pixel.
//
// This function is concurrent-safe.
func SetScreenSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	ui.SetScreenSize(width, height)
}

// SetScreenScale changes the scale of the screen.
//
// Note that the actual screen is multiplied not only by the given scale but also
// by the device scale on high-DPI display.
// If you pass inverse of the device scale,
// you can disable this automatical device scaling as a result.
// You can get the device scale by DeviceScaleFactor function.
//
// This function is concurrent-safe.
func SetScreenScale(scale float64) {
	if scale <= 0 {
		panic("ebiten: scale must be positive")
	}
	ui.SetScreenScale(scale)
}

// ScreenScale returns the current screen scale.
//
// If Run is not called, this returns 0.
//
// This function is concurrent-safe.
func ScreenScale() float64 {
	return ui.ScreenScale()
}

// IsCursorVisible returns a boolean value indicating whether
// the cursor is visible or not.
//
// IsCursorVisible always returns false on mobiles.
//
// This function is concurrent-safe.
func IsCursorVisible() bool {
	return ui.IsCursorVisible()
}

// SetCursorVisible changes the state of cursor visiblity.
//
// SetCursorVisible does nothing on mobiles.
//
// This function is concurrent-safe.
func SetCursorVisible(visible bool) {
	ui.SetCursorVisible(visible)
}

// SetCursorVisibility is deprecated as of 1.6.0-alpha. Use SetCursorVisible instead.
func SetCursorVisibility(visible bool) {
	SetCursorVisible(visible)
}

// IsFullscreen returns a boolean value indicating whether
// the current mode is fullscreen or not.
//
// IsFullscreen always returns false on mobiles.
//
// This function is concurrent-safe.
func IsFullscreen() bool {
	return ui.IsFullscreen()
}

// SetFullscreen changes the current mode to fullscreen or not.
//
// On fullscreen mode, the game screen is automatically enlarged
// to fit with the monitor. The current scale value is ignored.
//
// On desktops, Ebiten uses 'windowed' fullscreen mode, which doesn't change
// your monitor's resolution.
//
// On browsers, the game screen is resized to fit with the body element (client) size.
// Additionally, the game screen is automatically resized when the body element is resized.
// Note that this has nothing to do with 'screen' which is outside of 'window'.
// It is recommended to put Ebiten game in an iframe, and if you want to make the game 'fullscreen'
// on browsers with Fullscreen API, you can do this by applying the API to the iframe.
//
// SetFullscreen does nothing on mobiles.
//
// This function is concurrent-safe.
func SetFullscreen(fullscreen bool) {
	ui.SetFullscreen(fullscreen)
}

// IsRunnableInBackground returns a boolean value indicating whether
// the game runs even in background.
//
// This function is concurrent-safe.
func IsRunnableInBackground() bool {
	return ui.IsRunnableInBackground()
}

// SetWindowDecorated sets the state if the window is decorated.
//
// SetWindowDecorated works only on desktops.
// SetWindowDecorated does nothing on other platforms.
//
// SetWindowDecorated panics if SetWindowDecorated is called after Run.
//
// This function is concurrent-safe.
func SetWindowDecorated(decorated bool) {
	ui.SetWindowDecorated(decorated)
}

// IsWindowDecorated returns a boolean value indicating whether
// the window is decorated.
//
// This function is concurrent-safe.
func IsWindowDecorated() bool {
	return ui.IsWindowDecorated()
}

// SetRunnableInBackground sets the state if the game runs even in background.
//
// If the given value is true, the game runs in background e.g. when losing focus.
// The initial state is false.
//
// Known issue: On browsers, even if the state is on, the game doesn't run in background tabs.
// This is because browsers throttles background tabs not to often update.
//
// SetRunnableInBackground does nothing on mobiles so far.
//
// This function is concurrent-safe.
func SetRunnableInBackground(runnableInBackground bool) {
	ui.SetRunnableInBackground(runnableInBackground)
}

// SetWindowTitle sets the title of the window.
//
// SetWindowTitle does nothing on mobiles.
//
// SetWindowTitle is concurrent-safe.
func SetWindowTitle(title string) {
	ui.SetWindowTitle(title)
}

// SetWindowIcon sets the icon of the game window.
//
// If len(iconImages) is 0, SetWindowIcon reverts the icon to the default one.
//
// For desktops, see the document of glfwSetWindowIcon of GLFW 3.2:
//
//     This function sets the icon of the specified window.
//     If passed an array of candidate images, those of or closest to the sizes
//     desired by the system are selected.
//     If no images are specified, the window reverts to its default icon.
//
//     The desired image sizes varies depending on platform and system settings.
//     The selected images will be rescaled as needed.
//     Good sizes include 16x16, 32x32 and 48x48.
//
// As macOS windows don't have icons, SetWindowIcon doesn't work on macOS.
//
// SetWindowIcon doesn't work on browsers or mobiles.
//
// This function is concurrent-safe.
func SetWindowIcon(iconImages []image.Image) {
	ui.SetWindowIcon(iconImages)
}

// DeviceScaleFactor returns a device scale factor value.
//
// DeviceScaleFactor returns a meaningful value on high-DPI display environment,
// otherwise DeviceScaleFactor returns 1.
//
// DeviceScaleFactor might panic on init function on some devices like Android.
// Then, it is not recommended to call DeviceScaleFactor from init functions.
//
// DeviceScaleFactor is concurrent-safe.
func DeviceScaleFactor() float64 {
	return devicescale.DeviceScale()
}

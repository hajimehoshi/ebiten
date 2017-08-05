// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package ui

import "github.com/go-gl/glfw/v3.2/glfw"

var window *glfw.Window

func Keyboard() []rune {
	if window == nil {
		window = currentUI.window
	}
	if runebuffer == nil {
		runebuffer = make([]rune, 0, 1024)
		window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
			go func() {
				rblock.Lock()
				runebuffer = append(runebuffer, char)
				rblock.Unlock()
			}()
		})
	}
	rblock.Lock()
	rb := runebuffer
	runebuffer = runebuffer[:0]
	rblock.Unlock()
	if window != currentUI.window && currentUI.window != nil {
		window = currentUI.window
		rblock.Lock()
		runebuffer = nil
		rblock.Unlock()
	}
	return rb
}

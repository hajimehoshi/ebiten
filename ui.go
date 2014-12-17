/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"fmt"
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"image"
	"runtime"
)

var currentUI *ui

func init() {
	runtime.LockOSThread()

	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		panic(fmt.Sprintf("%v: %v\n", err, desc))
	})
	if !glfw.Init() {
		panic("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		panic(err)
	}

	currentUI = &ui{
		window: window,
		funcs:  make(chan func()),
	}
	currentUI.run()
	currentUI.use(func() {
		gl.Init()
		gl.Enable(gl.TEXTURE_2D)
		gl.Enable(gl.BLEND)
	})
}

type ui struct {
	window          *glfw.Window
	scale           int
	graphicsContext *graphicsContext
	input           input
	funcs           chan func()
}

func startUI(width, height, scale int, title string) error {
	monitor, err := glfw.GetPrimaryMonitor()
	if err != nil {
		return err
	}
	videoMode, err := monitor.GetVideoMode()
	if err != nil {
		return err
	}
	x := (videoMode.Width - width*scale) / 2
	y := (videoMode.Height - height*scale) / 3

	window := currentUI.window
	window.SetSize(width*scale, height*scale)
	window.SetTitle(title)
	window.SetPosition(x, y)
	window.Show()

	ui := currentUI
	ui.scale = scale

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width
	ui.use(func() {
		ui.graphicsContext, err = newGraphicsContext(width, height, realScale)
	})
	return err
}

func (u *ui) doEvents() {
	glfw.PollEvents()
	u.update()
}

func (u *ui) terminate() {
	glfw.Terminate()
}

func (u *ui) isClosed() bool {
	return u.window.ShouldClose()
}

func (u *ui) Sync(f func()) {
	u.use(f)
}

func (u *ui) draw(f func(GraphicsContext) error) (err error) {
	u.use(func() {
		u.graphicsContext.preUpdate()
	})
	if err = f(&syncGraphicsContext{
		syncer:               u,
		innerGraphicsContext: u.graphicsContext,
	}); err != nil {
		return
	}
	u.use(func() {
		u.graphicsContext.postUpdate()
		u.window.SwapBuffers()
	})
	return
}

func (u *ui) newTexture(img image.Image, filter int) (*Texture, error) {
	var texture *Texture
	var err error
	u.use(func() {
		texture, err = idsInstance.createTexture(img, filter)
	})
	return texture, err
}

func (u *ui) newRenderTarget(width, height int, filter int) (*RenderTarget, error) {
	var renderTarget *RenderTarget
	var err error
	u.use(func() {
		renderTarget, err = idsInstance.createRenderTarget(width, height, filter)
	})
	return renderTarget, err
}

func (u *ui) run() {
	go func() {
		runtime.LockOSThread()
		u.window.MakeContextCurrent()
		for f := range u.funcs {
			f()
		}
	}()
}

func (u *ui) use(f func()) {
	ch := make(chan struct{})
	u.funcs <- func() {
		f()
		close(ch)
	}
	<-ch
}

func (u *ui) update() {
	u.input.update(u.window, u.scale)
}

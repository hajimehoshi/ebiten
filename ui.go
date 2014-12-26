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
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/internal/opengl"
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
		opengl.Init()
		glfw.SwapInterval(1)
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

	ch := make(chan struct{})
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		close(ch)
	})
	for {
		done := false
		glfw.PollEvents()
		select {
		case <-ch:
			done = true
		default:
		}
		if done {
			break
		}
	}

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

func (u *ui) draw(f func(*Image) error) (err error) {
	u.use(func() {
		err = u.graphicsContext.preUpdate()
	})
	if err != nil {
		return
	}
	err = f(&Image{syncer: u, inner: u.graphicsContext.screen})
	if err != nil {
		return
	}
	u.use(func() {
		err = u.graphicsContext.postUpdate()
		if err != nil {
			return
		}
		u.window.SwapBuffers()
	})
	return
}

func (u *ui) newImageFromImage(img image.Image, filter Filter) (*Image, error) {
	var innerImage *innerImage
	var err error
	u.use(func() {
		var texture *opengl.Texture
		texture, err = opengl.NewTextureFromImage(img, opengl.Filter(filter))
		if err != nil {
			return
		}
		innerImage, err = newInnerImage(texture)
	})
	if err != nil {
		return nil, err
	}
	return &Image{syncer: u, inner: innerImage}, nil
}

func (u *ui) newImage(width, height int, filter Filter) (*Image, error) {
	var innerImage *innerImage
	var err error
	u.use(func() {
		var texture *opengl.Texture
		texture, err = opengl.NewTexture(width, height, opengl.Filter(filter))
		if err != nil {
			return
		}
		innerImage, err = newInnerImage(texture)
		innerImage.Clear()
	})
	if err != nil {
		return nil, err
	}
	return &Image{syncer: u, inner: innerImage}, nil
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
		defer close(ch)
		f()
	}
	<-ch
}

func (u *ui) update() {
	u.input.update(u.window, u.scale)
}

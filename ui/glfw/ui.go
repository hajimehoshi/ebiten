package glfw

import (
	"errors"
	"fmt"
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/opengl"
	"github.com/hajimehoshi/ebiten/input"
	"github.com/hajimehoshi/ebiten/ui"
)

func init() {
	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		panic(fmt.Sprintf("%v: %v\n", err, desc))
	})
}

type UI struct {
	canvas *canvas
}

func (u *UI) Start(width, height, scale int, title string) (ui.Canvas, error) {
	if !glfw.Init() {
		return nil, errors.New("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	window, err := glfw.CreateWindow(width*scale, height*scale, title, nil, nil)
	if err != nil {
		return nil, err
	}

	c := &canvas{
		window:    window,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}
	input.SetKeyboard(&c.keyboard)
	graphics.SetTextureFactory(c)

	c.run(width, height, scale)

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width
	c.use(func() {
		c.contextUpdater, err = opengl.Initialize(width, height, realScale)
	})
	if err != nil {
		return nil, err
	}

	u.canvas = c
	return c, nil
}

func (u *UI) DoEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *UI) Terminate() {
	glfw.Terminate()
}

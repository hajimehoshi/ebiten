package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/opengl"
	"github.com/hajimehoshi/ebiten/ui"
	"image"
	"runtime"
)

type Canvas struct {
	window    *glfw.Window
	context   *opengl.Context
	funcs     chan func()
	funcsDone chan struct{}
}

func NewCanvas(width, height, scale int, title string) *Canvas {
	window, err := glfw.CreateWindow(width*scale, height*scale, title, nil, nil)
	if err != nil {
		panic(err)
	}
	canvas := &Canvas{
		window:    window,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, windowHeight := window.GetFramebufferSize()
	realScale := windowWidth / width
	_ = windowHeight

	canvas.run()
	canvas.use(func() {
		canvas.context = opengl.NewContext(width, height, realScale)
	})
	return canvas
}

func (c *Canvas) Draw(f func(graphics.Context)) {
	c.use(func() {
		c.context.Update(f)
		c.window.SwapBuffers()
	})
}

func (c *Canvas) IsClosed() bool {
	return c.window.ShouldClose()
}

func (c *Canvas) InputState() ui.InputState {
	return &InputState{newKeys(), -1, -1}
}

func (c *Canvas) CreateTexture(img image.Image, filter graphics.Filter) (graphics.TextureId, error) {
	var id graphics.TextureId
	var err error
	c.use(func() {
		id, err = opengl.CreateTexture(img, filter)
	})
	return id, err
}

func (c *Canvas) CreateRenderTarget(width, height int, filter graphics.Filter) (graphics.RenderTargetId, error) {
	var id graphics.RenderTargetId
	var err error
	c.use(func() {
		id, err = opengl.CreateRenderTarget(width, height, filter)
	})
	return id, err
}

func (c *Canvas) run() {
	go func() {
		runtime.LockOSThread()
		c.window.MakeContextCurrent()
		glfw.SwapInterval(1)
		for {
			f := <-c.funcs
			f()
			c.funcsDone <- struct{}{}
		}
	}()
}

func (c *Canvas) use(f func()) {
	c.funcs <- f
	<-c.funcsDone
}

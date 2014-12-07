package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/opengl"
	"github.com/hajimehoshi/ebiten/input"
	"github.com/hajimehoshi/ebiten/ui"
	"image"
	"runtime"
)

type canvas struct {
	window    *glfw.Window
	context   *opengl.Context
	keyboard  *keyboard
	funcs     chan func()
	funcsDone chan struct{}
}

func newCanvas(width, height, scale int, title string) *canvas {
	window, err := glfw.CreateWindow(width*scale, height*scale, title, nil, nil)
	if err != nil {
		panic(err)
	}
	canvas := &canvas{
		window:    window,
		keyboard:  newKeyboard(),
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}

	input.SetKeyboard(canvas.keyboard)
	graphics.SetTextureFactory(canvas)

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width

	canvas.run()
	canvas.use(func() {
		canvas.context = opengl.NewContext(width, height, realScale)
	})
	return canvas
}

func (c *canvas) Draw(d ui.Drawer) (err error) {
	c.use(func() {
		err = c.context.Update(d)
		c.window.SwapBuffers()
	})
	return
}

func (c *canvas) IsClosed() bool {
	return c.window.ShouldClose()
}

func (c *canvas) NewTextureID(img image.Image, filter graphics.Filter) (graphics.TextureID, error) {
	var id graphics.TextureID
	var err error
	c.use(func() {
		id, err = opengl.NewTextureID(img, filter)
	})
	return id, err
}

func (c *canvas) NewRenderTargetID(width, height int, filter graphics.Filter) (graphics.RenderTargetID, error) {
	var id graphics.RenderTargetID
	var err error
	c.use(func() {
		id, err = opengl.NewRenderTargetID(width, height, filter)
	})
	return id, err
}

func (c *canvas) run() {
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

func (c *canvas) use(f func()) {
	c.funcs <- f
	<-c.funcsDone
}

func (c *canvas) update() {
	c.keyboard.update(c.window)
}

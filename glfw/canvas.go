package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/opengl"
	"image"
	"runtime"
)

type canvas struct {
	window    *glfw.Window
	context   *opengl.GraphicsContext
	keyboard  keyboard
	mouse     mouse
	funcs     chan func()
	funcsDone chan struct{}
}

func (c *canvas) Draw(d ebiten.Drawer2) (err error) {
	c.use(func() {
		c.context.PreUpdate()
	})
	if err = d.Draw(&context{c}); err != nil {
		return
	}
	c.use(func() {
		c.context.PostUpdate()
		c.window.SwapBuffers()
	})
	return
}

func (c *canvas) IsClosed() bool {
	return c.window.ShouldClose()
}

func (c *canvas) NewTextureID(img image.Image, filter ebiten.Filter) (ebiten.TextureID, error) {
	var id ebiten.TextureID
	var err error
	c.use(func() {
		id, err = opengl.NewTextureID(img, filter)
	})
	return id, err
}

func (c *canvas) NewRenderTargetID(width, height int, filter ebiten.Filter) (ebiten.RenderTargetID, error) {
	var id ebiten.RenderTargetID
	var err error
	c.use(func() {
		id, err = opengl.NewRenderTargetID(width, height, filter)
	})
	return id, err
}

func (c *canvas) run(width, height, scale int) {
	go func() {
		runtime.LockOSThread()
		c.window.MakeContextCurrent()
		glfw.SwapInterval(1)
		for {
			(<-c.funcs)()
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
	c.mouse.update(c.window)
}

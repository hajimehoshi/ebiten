package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/ui"
	"log"
)

func init() {
	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		log.Fatalf("%v: %v\n", err, desc)
	})
}

type UI struct {
	canvas *Canvas
}

func (u *UI) CreateCanvas(width, height, scale int, title string) ui.Canvas {
	if !glfw.Init() {
		panic("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	u.canvas = NewCanvas(width, height, scale, title)
	return u.canvas
}

func (u *UI) Start() {
}

func (u *UI) DoEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *UI) Terminate() {
	glfw.Terminate()
}

func (u *UI) TextureFactory() graphics.TextureFactory {
	return u.canvas
}

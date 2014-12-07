package glfw

import (
	"errors"
	glfw "github.com/go-gl/glfw3"
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

func (u *UI) Start(width, height, scale int, title string) (ui.Canvas, error) {
	if !glfw.Init() {
		return nil, errors.New("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	u.canvas = NewCanvas(width, height, scale, title)
	return u.canvas, nil
}

func (u *UI) DoEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *UI) Terminate() {
	glfw.Terminate()
}

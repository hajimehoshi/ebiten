// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

// #cgo LDFLAGS: -framework GLUT -framework OpenGL
//
// #include <stdlib.h>
// #include <GLUT/glut.h>
//
// void display(void);
// void motion(int x, int y);
// void idle(void);
//
// static void setGlutFuncs(void) {
//   glutDisplayFunc(display);
//   glutMotionFunc(motion);
//   glutIdleFunc(idle);
// }
//
import "C"
import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/example/game/blank"
	"github.com/hajimehoshi/go.ebiten/example/game/input"
	"github.com/hajimehoshi/go.ebiten/example/game/monochrome"
	"github.com/hajimehoshi/go.ebiten/example/game/rects"
	"github.com/hajimehoshi/go.ebiten/example/game/rotating"
	"github.com/hajimehoshi/go.ebiten/example/game/sprites"
	"github.com/hajimehoshi/go.ebiten/example/game/text"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/opengl"
	"os"
	"runtime"
	"time"
	"unsafe"
)

type GlutInputEvent struct {
	X int
	Y int
}

type GlutUI struct {
	glutInputting chan GlutInputEvent
	updating      chan chan func()
}

var currentUI *GlutUI

//export display
func display() {
	ch := make(chan func())
	currentUI.updating <- ch
	f := <-ch
	f()
	C.glutSwapBuffers()
}

//export motion
func motion(x, y C.int) {
	currentUI.glutInputting <- GlutInputEvent{
		X: int(x),
		Y: int(y),
	}
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func NewGlutUI(screenWidth, screenHeight, screenScale int) *GlutUI {
	ui := &GlutUI{
		glutInputting: make(chan GlutInputEvent, 10),
		updating:      make(chan chan func()),
	}

	cargs := []*C.char{}
	for _, arg := range os.Args {
		cargs = append(cargs, C.CString(arg))
	}
	defer func() {
		for _, carg := range cargs {
			C.free(unsafe.Pointer(carg))
		}
	}()
	cargc := C.int(len(cargs))

	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA)
	C.glutInitWindowSize(
		C.int(screenWidth*screenScale),
		C.int(screenHeight*screenScale))

	title := C.CString("Ebiten Demo")
	defer C.free(unsafe.Pointer(title))
	C.glutCreateWindow(title)

	C.setGlutFuncs()

	return ui
}

func (ui *GlutUI) Run() {
	C.glutMainLoop()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	gameName := ""
	if 2 <= len(os.Args) {
		gameName = os.Args[1]
	}

	var game ebiten.Game
	switch gameName {
	case "blank":
		game = blank.New()
	case "input":
		game = input.New()
	case "monochrome":
		game = monochrome.New()
	case "rects":
		game = rects.New()
	case "rotating":
		game = rotating.New()
	case "sprites":
		game = sprites.New()
	case "text":
		game = text.New()
	default:
		game = rotating.New()
	}

	const screenScale = 2
	screenWidth := game.ScreenWidth()
	screenHeight := game.ScreenHeight()
	currentUI = NewGlutUI(screenWidth, screenHeight, screenScale)

	graphicsDevice := opengl.NewDevice(
		screenWidth, screenHeight, screenScale,
		currentUI.updating)

	game.Init(graphicsDevice.TextureFactory())
	draw := graphicsDevice.Drawing()

	input := make(chan ebiten.InputState)
	go func() {
		ch := currentUI.glutInputting
		for {
			event := <-ch
			x := event.X / screenScale
			y := event.Y / screenScale
			if x < 0 {
				x = 0
			} else if screenWidth <= x {
				x = screenWidth - 1
			}
			if y < 0 {
				y = 0
			} else if screenHeight <= y {
				y = screenHeight - 1
			}
			input <- ebiten.InputState{
				X: x,
				Y: y,
			}
		}
	}()

	go func() {
		frameTime := time.Duration(
			int64(time.Second) / int64(game.Fps()))
		update := time.Tick(frameTime)
		for {
			select {
			case <-input:
			case <-update:
				game.Update()
			case drawing := <-draw:
				ch := make(chan interface{})
				drawing <- func(g graphics.Context, offscreen graphics.Texture) {
					game.Draw(g, offscreen)
					close(ch)
				}
				<-ch
			}
		}
	}()

	currentUI.Run()
}

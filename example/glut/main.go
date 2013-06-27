package main

// #cgo LDFLAGS: -framework GLUT -framework OpenGL
//
// #include <stdlib.h>
// #include <GLUT/glut.h>
//
// void display(void);
// void mouse(int button, int state, int x, int y);
// void idle(void);
//
// static void setGlutFuncs(void) {
//   glutDisplayFunc(display);
//   glutMouseFunc(mouse);
//   glutIdleFunc(idle);
// }
//
import "C"
import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/example/game/blank"
	"github.com/hajimehoshi/go.ebiten/example/game/monochrome"
	"github.com/hajimehoshi/go.ebiten/example/game/rects"
	"github.com/hajimehoshi/go.ebiten/example/game/rotating"
	"github.com/hajimehoshi/go.ebiten/example/game/sprites"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/opengl"
	"os"
	"runtime"
	"time"
	"unsafe"
)

type GlutInputEventState int

const (
	GlutInputEventStateUp GlutInputEventState = iota
	GlutInputEventStateDown
)

type GlutInputEvent struct {
	State GlutInputEventState
	X     int
	Y     int
}

type GlutUI struct {
	screenScale      int
	glutInputEventCh chan GlutInputEvent
	updating         chan chan func()
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

//export mouse
func mouse(button, glutState, x, y C.int) {
	var state GlutInputEventState
	switch glutState {
	case C.GLUT_UP:
		state = GlutInputEventStateUp
	case C.GLUT_DOWN:
		state = GlutInputEventStateDown
	default:
		panic("invalid glutState")
	}
	currentUI.glutInputEventCh <- GlutInputEvent{
		State: state,
		X:     int(x),
		Y:     int(y),
	}
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func NewGlutUI(screenWidth, screenHeight, screenScale int) *GlutUI {
	ui := &GlutUI{
		screenScale:      screenScale,
		glutInputEventCh: make(chan GlutInputEvent, 10),
		updating:         make(chan chan func()),
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
	case "monochrome":
		game = monochrome.New()
	case "rects":
		game = rects.New()
	case "rotating":
		game = rotating.New()
	case "sprites":
		game = sprites.New()
	default:
		game = rotating.New()
	}

	screenScale := 2
	currentUI = NewGlutUI(game.ScreenWidth(), game.ScreenHeight(), screenScale)

	input := make(chan ebiten.InputState)
	go func() {
		ch := currentUI.glutInputEventCh
		var inputState ebiten.InputState
		for {
			event := <-ch
			switch event.State {
			case GlutInputEventStateUp:
				inputState.IsTapped = false
				inputState.X = 0
				inputState.Y = 0
			case GlutInputEventStateDown:
				inputState.IsTapped = true
				inputState.X = event.X
				inputState.Y = event.Y
			}
			input <- inputState
		}
	}()

	graphicsDevice := opengl.NewDevice(
		game.ScreenWidth(), game.ScreenHeight(), screenScale,
		currentUI.updating)

	game.Init(graphicsDevice.TextureFactory())
	draw := graphicsDevice.Drawing()

	go func() {
		frameTime := time.Duration(int64(time.Second) / int64(game.Fps()))
		update := time.Tick(frameTime)
		inputState := ebiten.InputState{}
		for {
			select {
			case inputState = <-input:
			case <-update:
				game.Update(inputState)
				inputState = ebiten.InputState{}
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

type FuncInitializer struct {
	f func(graphics.TextureFactory)
}

func (i *FuncInitializer) Initialize(tf graphics.TextureFactory) {
	i.f(tf)
}

type FuncDrawable struct {
	f func(graphics.Context, graphics.Texture)
}

func (d *FuncDrawable) Draw(g graphics.Context, offscreen graphics.Texture) {
	d.f(g, offscreen)
}

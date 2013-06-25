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
	"os"
	"runtime"
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
	device           graphics.Device
	glutInputEventCh chan GlutInputEvent
}

var currentUI *GlutUI

//export display
func display() {
	currentUI.device.Update()
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

func (ui *GlutUI) Init(screenWidth, screenHeight, screenScale int) {
	ui.screenScale = screenScale
	ui.glutInputEventCh = make(chan GlutInputEvent, 10)

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
}

func (ui *GlutUI) Run(device graphics.Device) {
	ui.device = device
	C.glutMainLoop()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	gameName := ""
	if 2 <= len(os.Args) {
		gameName = os.Args[1]
	}

	var gm ebiten.Game
	switch gameName {
	case "blank":
		gm = blank.New()
	case "monochrome":
		gm = monochrome.New()
	case "rects":
		gm = rects.New()
	case "rotating":
		gm = rotating.New()
	case "sprites":
		gm = sprites.New()
	default:
		gm = rotating.New()
	}

	screenScale := 2
	currentUI = &GlutUI{}
	currentUI.Init(gm.ScreenWidth(), gm.ScreenHeight(), screenScale)

	input := make(chan ebiten.InputState)
	go func() {
		ch := currentUI.glutInputEventCh
		var inputState ebiten.InputState
		for {
			select {
			case event := <-ch:
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
			default:
				// do nothing
			}
			input <- inputState
		}
	}()

	ebiten.OpenGLRun(gm, currentUI, screenScale, input)
}

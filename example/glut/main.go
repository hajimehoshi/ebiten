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
	"github.com/hajimehoshi/go.ebiten/example/game/monochrome"
	"github.com/hajimehoshi/go.ebiten/example/game/rects"
	"github.com/hajimehoshi/go.ebiten/example/game/rotating"
	"github.com/hajimehoshi/go.ebiten/example/game/sprites"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"os"
	"runtime"
	"unsafe"
)

type GlutUI struct {
	device graphics.Device
}

var currentUI *GlutUI

//export display
func display() {
	currentUI.device.Update()
	C.glutSwapBuffers()
}

//export mouse
func mouse(button, state, x, y C.int) {
	
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func (ui *GlutUI) Init(screenWidth, screenHeight, screenScale int) {
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

	ebiten.OpenGLRun(gm, currentUI, screenScale)
}

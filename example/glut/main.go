package main

// #cgo LDFLAGS: -framework GLUT -framework OpenGL
//
// #include <stdlib.h>
// #include <GLUT/glut.h>
//
// void display(void);
// void idle(void);
//
// static void setGlutFuncs(void) {
//   glutDisplayFunc(display);
//   glutIdleFunc(idle);
// }
//
import "C"
import (
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/example/game"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"os"
	"runtime"
	"unsafe"
)

type GlutUI struct {
	screenWidth  int
	screenHeight int
	screenScale  int
	device       graphics.Device
}

var currentUI *GlutUI

//export display
func display() {
	currentUI.device.Update()
	C.glutSwapBuffers()
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func (ui *GlutUI) Init() {
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

	ui.screenWidth = 256
	ui.screenHeight = 240
	ui.screenScale = 2

	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA)
	C.glutInitWindowSize(
		C.int(ui.screenWidth*ui.screenScale),
		C.int(ui.screenHeight*ui.screenScale))

	title := C.CString("Ebiten Demo")
	defer C.free(unsafe.Pointer(title))
	C.glutCreateWindow(title)

	C.setGlutFuncs()
}

func (ui *GlutUI) ScreenWidth() int {
	return ui.screenWidth
}

func (ui *GlutUI) ScreenHeight() int {
	return ui.screenHeight
}

func (ui *GlutUI) ScreenScale() int {
	return ui.screenScale
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
	case "sprites":
		gm = game.NewSprites()
	default:
		gm = game.NewRotatingImage()
	}
	currentUI = &GlutUI{}
	currentUI.Init()

	ebiten.OpenGLRun(gm, currentUI)
}

package main

// #cgo LDFLAGS: -framework GLUT -framework OpenGL
//
// #include <stdlib.h>
// #include <GLUT/glut.h>
//
// void display(void);
// void idle(void);
//
// static void setDisplayFunc(void) {
//   glutDisplayFunc(display);
// }
//
// static void setIdleFunc(void) {
//   glutIdleFunc(idle);
// }
//
import "C"
import (
	"image/color"
	"os"
	"unsafe"
	"github.com/hajimehoshi/go-ebiten/graphics"
)

var device *graphics.Device

type DemoGame struct {
}

func (game *DemoGame) Update() {
}

func (game *DemoGame) Draw(g *graphics.GraphicsContext, offscreen *graphics.Texture) {
	g.Fill(&color.RGBA{R: 0, G: 0, B: 255, A: 255})
}

//export display
func display() {
	device.Update()
	C.glutSwapBuffers()
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func main() {
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

	screenWidth  := 256
	screenHeight := 240
	screenScale  := 2

	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA);
	C.glutInitWindowSize(C.int(screenWidth * screenScale),
		C.int(screenHeight * screenScale))

	title := C.CString("Ebiten Demo")
	defer C.free(unsafe.Pointer(title))
	C.glutCreateWindow(title)

	C.setDisplayFunc()
	C.setIdleFunc()

	game := &DemoGame{}
	device = graphics.NewDevice(screenWidth, screenHeight, screenScale,
		func(g *graphics.GraphicsContext, offscreen *graphics.Texture) {
			game.Draw(g, offscreen)
		})

	C.glutMainLoop()
}

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
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
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

type DemoGame struct {
	ebitenTexture graphics.Texture
	x             int
}

func (game *DemoGame) Init(tf graphics.TextureFactory) {
	file, err := os.Open("ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	game.ebitenTexture = tf.NewTextureFromImage(img)
}

func (game *DemoGame) Update() {
	game.x++
}

func (game *DemoGame) Draw(g graphics.GraphicsContext, offscreen graphics.TextureID) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx, ty := float64(game.ebitenTexture.Width), float64(game.ebitenTexture.Height)
	geometryMatrix.Translate(-tx/2, -ty/2)
	geometryMatrix.Rotate(float64(game.x) / 60)
	geometryMatrix.Translate(tx/2, ty/2)
	geometryMatrix.Translate(100, 100)
	g.DrawTexture(game.ebitenTexture.ID,
		0, 0, int(tx), int(ty),
		geometryMatrix,
		matrix.IdentityColor())
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	game := &DemoGame{}
	currentUI = &GlutUI{}
	currentUI.Init()

	ebiten.OpenGLRun(game, currentUI)

}

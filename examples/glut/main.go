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
	"image"
	"image/color"
	_ "image/png"
	"os"
	"unsafe"
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type GlutUI struct{
	screenWidth int
	screenHeight int
	screenScale int
	device graphics.Device
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

	ui.screenWidth  = 256
	ui.screenHeight = 240
	ui.screenScale  = 2

	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA);
	C.glutInitWindowSize(
		C.int(ui.screenWidth  * ui.screenScale),
		C.int(ui.screenHeight * ui.screenScale))

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
	x int
}

func (game *DemoGame) Update() {
	if game.ebitenTexture == nil {
		file, err := os.Open("ebiten.png")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		
		img, _, err := image.Decode(file)
		if err != nil {
			panic(err)
		}

		// TODO: It looks strange to get a texture from the device.
		game.ebitenTexture = currentUI.device.NewTextureFromImage(img)
	}
	game.x++
}

func (game *DemoGame) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})
	if game.ebitenTexture == nil {
		return
	}
	geometryMatrix := graphics.IdentityGeometryMatrix()
	geometryMatrix.SetTx(float64(game.x))
	geometryMatrix.SetTy(float64(game.x))
	g.DrawTexture(game.ebitenTexture,
		0, 0, game.ebitenTexture.Width(), game.ebitenTexture.Height(),
		geometryMatrix,
		graphics.IdentityColorMatrix())
}

func main() {
	game := &DemoGame{}
	currentUI = &GlutUI{}
	currentUI.Init()

	ebiten.OpenGLRun(game, currentUI)
}

// This package is experimental.
package glut

// #cgo LDFLAGS: -framework GLUT -framework OpenGL
//
// #include <stdlib.h>
// #include <GLUT/glut.h>
//
// void display(void);
// void mouse(int button, int state, int x, int y);
// void motion(int x, int y);
// void idle(void);
//
// static void setGlutFuncs(void) {
//   glutDisplayFunc(display);
//   glutMouseFunc(mouse);
//   glutMotionFunc(motion);
//   glutIdleFunc(idle);
// }
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"os"
	"unsafe"
)

type glutInputEvent struct {
	IsActive bool
	X        int
	Y        int
}

type UI struct {
	screenWidth    int
	screenHeight   int
	screenScale    int
	title          string
	initializing   chan ebiten.Game
	initialized    chan ebiten.Game
	updating       chan ebiten.Game
	updated        chan ebiten.Game
	input          chan ebiten.InputState
	graphicsDevice *opengl.Device
	glutInputting  chan glutInputEvent
}

var currentUI *UI

func New(screenWidth, screenHeight, screenScale int, title string) *UI {
	if currentUI != nil {
		panic("UI can't be duplicated.")
	}
	ui := &UI{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
		title:        title,
		initializing: make(chan ebiten.Game),
		initialized:  make(chan ebiten.Game),
		updating:     make(chan ebiten.Game),
		updated:      make(chan ebiten.Game),
		input:        make(chan ebiten.InputState),
		glutInputting: make(chan glutInputEvent),
	}
	currentUI = ui
	return ui
}

func (ui *UI) MainLoop() {
	cargs := []*C.char{C.CString(os.Args[0])}
	defer func() {
		for _, carg := range cargs {
			C.free(unsafe.Pointer(carg))
		}
	}()
	cargc := C.int(len(cargs))

	// Initialize OpenGL
	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA)
	C.glutInitWindowSize(
		C.int(ui.screenWidth*ui.screenScale),
		C.int(ui.screenHeight*ui.screenScale))

	cTitle := C.CString(ui.title)
	defer C.free(unsafe.Pointer(cTitle))
	C.glutCreateWindow(cTitle)

	ui.graphicsDevice = opengl.NewDevice(
		ui.screenWidth, ui.screenHeight, ui.screenScale)
	ui.graphicsDevice.Init()

	game := <-ui.initializing
	game.Init(ui.graphicsDevice.TextureFactory())
	ui.initialized <- game

	// Set the callbacks
	C.setGlutFuncs()

	C.glutMainLoop()
}

func (ui *UI) ScreenWidth() int {
	return ui.screenWidth
}

func (ui *UI) ScreenHeight() int {
	return ui.screenHeight
}

func (ui *UI) Initializing() chan<- ebiten.Game {
	return ui.initializing
}

func (ui *UI) Initialized() <-chan ebiten.Game {
	return ui.initialized
}

func (ui *UI) Updating() chan<- ebiten.Game {
	return ui.updating
}

func (ui *UI) Updated() <-chan ebiten.Game {
	return ui.updated
}

func (ui *UI) Input() <-chan ebiten.InputState {
	return ui.input
}

func (ui *UI) normalizePoint(x, y int) (newX, newY int) {
	x /= ui.screenScale
	y /= ui.screenScale
	if x < 0 {
		x = 0
	} else if ui.screenWidth <= x {
		x = ui.screenWidth - 1
	}
	if y < 0 {
		y = 0
	} else if ui.screenHeight <= y {
		y = ui.screenHeight - 1
	}
	return x, y
}

//export display
func display() {
	game := <-currentUI.updating
	currentUI.graphicsDevice.Update(game.Draw)
	currentUI.updated <- game
	C.glutSwapBuffers()
}

//export mouse
func mouse(button, state, x, y C.int) {
	if state != C.GLUT_DOWN {
		currentUI.input <- ebiten.InputState{-1, -1}
		return
	}
	newX, newY := currentUI.normalizePoint(int(x), int(y))
	currentUI.input <- ebiten.InputState{newX, newY}
}

//export motion
func motion(x, y C.int) {
	newX, newY := currentUI.normalizePoint(int(x), int(y))
	currentUI.input <- ebiten.InputState{newX, newY}
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

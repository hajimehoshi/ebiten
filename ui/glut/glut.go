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
	"github.com/hajimehoshi/go.ebiten"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/opengl"
	"os"
	"time"
	"unsafe"
)

type glutInputEvent struct {
	IsActive bool
	X        int
	Y        int
}

type GlutUI struct {
	screenScale    int
	glutInputting  chan glutInputEvent
	graphicsDevice *opengl.Device
	updating       chan func(graphics.Context)
	updated        chan bool
}

var currentUI *GlutUI

//export display
func display() {
	draw := <-currentUI.updating
	currentUI.graphicsDevice.Update(draw)
	currentUI.updated <- true
	C.glutSwapBuffers()
}

//export mouse
func mouse(button, state, x, y C.int) {
	event := glutInputEvent{false, 0, 0}
	if state == C.GLUT_DOWN {
		event.IsActive = true
		event.X = int(x)
		event.Y = int(y)
	}
	currentUI.glutInputting <- event
}

//export motion
func motion(x, y C.int) {
	currentUI.glutInputting <- glutInputEvent{
		IsActive: true,
		X:        int(x),
		Y:        int(y),
	}
}

//export idle
func idle() {
	C.glutPostRedisplay()
}

func new(screenWidth, screenHeight, screenScale int, title string) *GlutUI {
	ui := &GlutUI{
		glutInputting: make(chan glutInputEvent, 10),
		updating:      make(chan func(graphics.Context)),
		updated:       make(chan bool),
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

	// Initialize OpenGL
	C.glutInit(&cargc, &cargs[0])
	C.glutInitDisplayMode(C.GLUT_RGBA)
	C.glutInitWindowSize(
		C.int(screenWidth*screenScale),
		C.int(screenHeight*screenScale))

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.glutCreateWindow(cTitle)

	return ui
}

func Run(game ebiten.Game, screenWidth, screenHeight, screenScale int, title string) {
	ui := new(screenWidth, screenHeight, screenScale, title)
	currentUI = ui

	graphicsDevice := opengl.NewDevice(
		screenWidth, screenHeight, screenScale)
	ui.graphicsDevice = graphicsDevice
	graphicsDevice.Init()

	game.Init(ui.graphicsDevice.TextureFactory())

	input := make(chan ebiten.InputState)
	go func() {
		ch := ui.glutInputting
		for {
			event := <-ch
			inputState := ebiten.InputState{-1, -1}
			if event.IsActive {
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
				inputState.X = x
				inputState.Y = y
			}
			input <- inputState
		}
	}()

	go func() {
		frameTime := time.Duration(
			int64(time.Second) / int64(ebiten.FPS))
		tick := time.Tick(frameTime)
		gameContext := &GameContext{
			screenWidth:  screenWidth,
			screenHeight: screenHeight,
			inputState:   ebiten.InputState{-1, -1},
		}
		for {
			select {
			case gameContext.inputState = <-input:
			case <-tick:
				game.Update(gameContext)
			case ui.updating <- game.Draw:
				<-ui.updated
			}
		}
		os.Exit(0)
	}()

	// Set the callbacks
	C.setGlutFuncs()

	C.glutMainLoop()
}

type GameContext struct {
	screenWidth  int
	screenHeight int
	inputState   ebiten.InputState
}

func (context *GameContext) ScreenWidth() int {
	return context.screenWidth
}

func (context *GameContext) ScreenHeight() int {
	return context.screenHeight
}

func (context *GameContext) InputState() ebiten.InputState {
	return context.inputState
}

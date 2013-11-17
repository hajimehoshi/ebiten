package cocoa

// #cgo CFLAGS: -x objective-c -fobjc-arc
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework QuartzCore
//
// #include <stdlib.h>
// #include "input.h"
//
// void Run(size_t width, size_t height, size_t scale, const char* title);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"sync"
	"time"
	"unsafe"
)

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

type UI struct {
	screenWidth               int
	screenHeight              int
	screenScale               int
	title                     string
	updating                  chan ebiten.Game
	updated                   chan ebiten.Game
	input                     chan ebiten.InputState
	graphicsDevice            *opengl.Device
	funcsExecutedOnMainThread []func() // TODO: map?
	lock                      sync.Mutex
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
		updating:     make(chan ebiten.Game),
		updated:      make(chan ebiten.Game),
		input:        make(chan ebiten.InputState),
		funcsExecutedOnMainThread: []func(){},
	}
	currentUI = ui
	return ui
}

func (ui *UI) gameMainLoop(game ebiten.Game) {
	frameTime := time.Duration(int64(time.Second) / int64(ebiten.FPS))
	tick := time.Tick(frameTime)
	gameContext := &GameContext{
		screenWidth:  ui.screenWidth,
		screenHeight: ui.screenHeight,
		inputState:   ebiten.InputState{-1, -1},
	}
	ui.InitializeGame(game)
	for {
		select {
		case gameContext.inputState = <-ui.input:
		case <-tick:
			game.Update(gameContext)
		case ui.updating <- game:
			game = <-ui.updated
		}
	}
}

func (ui *UI) Run(game ebiten.Game) {
	go ui.gameMainLoop(game)

	cTitle := C.CString(ui.title)
	defer C.free(unsafe.Pointer(cTitle))

	C.Run(C.size_t(ui.screenWidth),
		C.size_t(ui.screenHeight),
		C.size_t(ui.screenScale),
		cTitle)
}

func (ui *UI) InitializeGame(game ebiten.Game) {
	ui.lock.Lock()
	defer ui.lock.Unlock()
	ui.funcsExecutedOnMainThread = append(ui.funcsExecutedOnMainThread, func() {
		game.InitTextures(ui.graphicsDevice.TextureFactory())
	})
}

func (ui *UI) DrawGame(game ebiten.Game) {
	ui.lock.Lock()
	defer ui.lock.Unlock()
	ui.funcsExecutedOnMainThread = append(ui.funcsExecutedOnMainThread, func() {
		ui.graphicsDevice.Update(game.Draw)
	})
}

//export ebiten_EbitenOpenGLView_Initialized
func ebiten_EbitenOpenGLView_Initialized() {
	if currentUI.graphicsDevice != nil {
		panic("The graphics device is already initialized")
	}

	currentUI.graphicsDevice = opengl.NewDevice(
		currentUI.screenWidth,
		currentUI.screenHeight,
		currentUI.screenScale)
	currentUI.graphicsDevice.Init()
}

//export ebiten_EbitenOpenGLView_Updating
func ebiten_EbitenOpenGLView_Updating() {
	currentUI.lock.Lock()
	defer currentUI.lock.Unlock()
	for _, f := range currentUI.funcsExecutedOnMainThread {
		f()
	}
	currentUI.funcsExecutedOnMainThread = currentUI.funcsExecutedOnMainThread[0:0]

	game := <-currentUI.updating
	currentUI.graphicsDevice.Update(game.Draw)
	currentUI.updated <- game
}

//export ebiten_EbitenOpenGLView_InputUpdated
func ebiten_EbitenOpenGLView_InputUpdated(inputType C.InputType, cx, cy C.int) {
	if inputType == C.InputTypeMouseUp {
		currentUI.input <- ebiten.InputState{-1, -1}
		return
	}

	x, y := int(cx), int(cy)
	x /= currentUI.screenScale
	y /= currentUI.screenScale
	if x < 0 {
		x = 0
	} else if currentUI.screenWidth <= x {
		x = currentUI.screenWidth - 1
	}
	if y < 0 {
		y = 0
	} else if currentUI.screenHeight <= y {
		y = currentUI.screenHeight - 1
	}
	currentUI.input <- ebiten.InputState{x, y}
}

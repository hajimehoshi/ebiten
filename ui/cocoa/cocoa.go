package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework QuartzCore
// 
// #include <stdlib.h>
// #include "input.h"
//
// void Start(size_t width, size_t height, size_t scale, const char* title);
// void WaitEvents(void);
// void BeginDrawing(void);
// void EndDrawing(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

func init() {
	runtime.LockOSThread()
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

type UI struct {
	screenWidth       int
	screenHeight      int
	screenScale       int
	title             string
	updating          chan struct{}
	updated           chan struct{}
	input             chan ebiten.InputState
	graphicsDevice    *opengl.Device
	lock              sync.Mutex
	gameContext       *GameContext
}

var currentUI *UI

func New(screenWidth, screenHeight, screenScale int, title string) *UI {
	if currentUI != nil {
		panic("UI can't be duplicated.")
	}
	ui := &UI{
		screenWidth:       screenWidth,
		screenHeight:      screenHeight,
		screenScale:       screenScale,
		title:             title,
		updating:          make(chan struct{}),
		updated:           make(chan struct{}),
		input:             make(chan ebiten.InputState),
		gameContext: &GameContext{
			screenWidth:  screenWidth,
			screenHeight: screenHeight,
			inputState:   ebiten.InputState{-1, -1},
		},
	}
	currentUI = ui
	return ui
}

func (ui *UI) gameMainLoop(game ebiten.Game) {
	frameTime := time.Duration(int64(time.Second) / int64(ebiten.FPS))
	tick := time.Tick(frameTime)
	for {
		select {
		case ui.gameContext.inputState = <-ui.input:
		case <-tick:
			game.Update(ui.gameContext)
		case ui.updating <- struct{}{}:
			//ui.DrawGame(game)
			<-ui.updated
		}
	}
}

func (ui *UI) Start(game ebiten.Game) {
	go ui.gameMainLoop(game)

	cTitle := C.CString(ui.title)
	defer C.free(unsafe.Pointer(cTitle))

	C.Start(C.size_t(ui.screenWidth),
		C.size_t(ui.screenHeight),
		C.size_t(ui.screenScale),
		cTitle)
	C.WaitEvents()
}

func (ui *UI) WaitEvents() {
	C.WaitEvents()
}

func (ui *UI) InitTextures(game ebiten.Game) {
	C.BeginDrawing()
	game.InitTextures(ui.graphicsDevice.TextureFactory())
	C.EndDrawing()
}

func (ui *UI) Draw(f func(graphics.Context)) {
	C.BeginDrawing()
	ui.graphicsDevice.Update(f)
	C.EndDrawing()
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
}

//export ebiten_EbitenOpenGLView_Updating
func ebiten_EbitenOpenGLView_Updating() {
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

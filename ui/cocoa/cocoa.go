package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// #include <stdlib.h>
// #include "input.h"
//
// void Start(size_t width, size_t height, size_t scale, const char* title);
// void PollEvents(void);
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
	screenWidth    int
	screenHeight   int
	screenScale    int
	title          string
	graphicsDevice *opengl.Device
	lock           sync.Mutex
	gameContext    *GameContext
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
		gameContext: &GameContext{
			screenWidth:  screenWidth,
			screenHeight: screenHeight,
			inputState:   ebiten.InputState{-1, -1},
		},
	}
	currentUI = ui
	return ui
}

func (ui *UI) Start() {
	cTitle := C.CString(ui.title)
	defer C.free(unsafe.Pointer(cTitle))

	C.Start(C.size_t(ui.screenWidth),
		C.size_t(ui.screenHeight),
		C.size_t(ui.screenScale),
		cTitle)
	C.PollEvents()
}

func (ui *UI) PollEvents() {
	C.PollEvents()
}

func (ui *UI) InitTextures(f func(graphics.TextureFactory)) {
	C.BeginDrawing()
	f(ui.graphicsDevice.TextureFactory())
	C.EndDrawing()
}

func (ui *UI) Update(f func(ebiten.GameContext)) {
	f(ui.gameContext)
}

func (ui *UI) Draw(f func(graphics.Context)) {
	C.BeginDrawing()
	ui.graphicsDevice.Update(f)
	C.EndDrawing()
}

//export ebiten_Initialized
func ebiten_Initialized() {
	if currentUI.graphicsDevice != nil {
		panic("The graphics device is already initialized")
	}
	currentUI.graphicsDevice = opengl.NewDevice(
		currentUI.screenWidth,
		currentUI.screenHeight,
		currentUI.screenScale)
}

//export ebiten_InputUpdated
func ebiten_InputUpdated(inputType C.InputType, cx, cy C.int) {
	if inputType == C.InputTypeMouseUp {
		currentUI.gameContext.inputState = ebiten.InputState{-1, -1}
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
	currentUI.gameContext.inputState = ebiten.InputState{x, y}
}

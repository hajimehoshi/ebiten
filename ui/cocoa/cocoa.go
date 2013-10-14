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
	"unsafe"
)

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
	}
	currentUI = ui
	return ui
}

func (ui *UI) MainLoop() {
	cTitle := C.CString(ui.title)
	defer C.free(unsafe.Pointer(cTitle))

	C.Run(C.size_t(ui.screenWidth),
		C.size_t(ui.screenHeight),
		C.size_t(ui.screenScale),
		cTitle)
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

	game := <-currentUI.initializing
	game.Init(currentUI.graphicsDevice.TextureFactory())
	currentUI.initialized <- game
}

//export ebiten_EbitenOpenGLView_Updating
func ebiten_EbitenOpenGLView_Updating() {
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

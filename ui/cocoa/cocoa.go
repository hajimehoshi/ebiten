package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// #include <stdlib.h>
// #include "input.h"
//
// void StartApplication(void);
// void* CreateGLContext(void* sharedGLContext);
// void SetCurrentGLContext(void* glContext);
// void* CreateWindow(size_t width, size_t height, const char* title, void* sharedGLContext);
// void PollEvents(void);
// void BeginDrawing(void* window);
// void EndDrawing(void* window);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"runtime"
	"unsafe"
)

func init() {
	runtime.LockOSThread()
}

type GameContext struct {
	screenWidth  int
	screenHeight int
}

func (context *GameContext) ScreenWidth() int {
	return context.screenWidth
}

func (context *GameContext) ScreenHeight() int {
	return context.screenHeight
}

type UI struct {
	screenWidth               int
	screenHeight              int
	screenScale               int
	graphicsDevice            *opengl.Device
	gameContext               *GameContext
	window                    unsafe.Pointer
	inputStateUpdatedChs      chan chan ebiten.InputStateUpdatedEvent
	inputStateUpdatedNotified chan ebiten.InputStateUpdatedEvent
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
		gameContext: &GameContext{
			screenWidth:  screenWidth,
			screenHeight: screenHeight,
		},
		inputStateUpdatedChs:      make(chan chan ebiten.InputStateUpdatedEvent),
		inputStateUpdatedNotified: make(chan ebiten.InputStateUpdatedEvent),
	}

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	C.StartApplication()

	context := C.CreateGLContext(unsafe.Pointer(nil))
	C.SetCurrentGLContext(context)
	ui.graphicsDevice = opengl.NewDevice(
		ui.screenWidth,
		ui.screenHeight,
		ui.screenScale)

	ui.window = C.CreateWindow(C.size_t(ui.screenWidth*ui.screenScale),
		C.size_t(ui.screenHeight*ui.screenScale),
		cTitle,
		context)
	currentUI = ui

	go ui.chLoop()

	return ui
}

func (ui *UI) chLoop() {
	inputStateUpdated := []chan ebiten.InputStateUpdatedEvent{}
	for {
		select {
		case ch := <-ui.inputStateUpdatedChs:
			inputStateUpdated = append(inputStateUpdated, ch)
		case e := <-ui.inputStateUpdatedNotified:
			for _, ch := range inputStateUpdated {
				ch <- e
			}
		}
	}
}

func (ui *UI) PollEvents() {
	C.PollEvents()
}

func (ui *UI) InitTextures(f func(graphics.TextureFactory)) {
	C.BeginDrawing(ui.window)
	f(ui.graphicsDevice.TextureFactory())
	C.EndDrawing(ui.window)
}

func (ui *UI) Update(f func(ebiten.GameContext)) {
	f(ui.gameContext)
}

func (ui *UI) Draw(f func(graphics.Canvas)) {
	C.BeginDrawing(ui.window)
	ui.graphicsDevice.Update(f)
	C.EndDrawing(ui.window)
}

func (ui *UI) InputStateUpdated() <-chan ebiten.InputStateUpdatedEvent {
	ch := make(chan ebiten.InputStateUpdatedEvent)
	go func() {
		ui.inputStateUpdatedChs <- ch
	}()
	return ch
}

func (ui *UI) notifyInputStateUpdated(e ebiten.InputStateUpdatedEvent) {
	go func() {
		ui.inputStateUpdatedNotified <- e
	}()
}

//export ebiten_InputUpdated
func ebiten_InputUpdated(inputType C.InputType, cx, cy C.int) {
	ui := currentUI

	if inputType == C.InputTypeMouseUp {
		e := ebiten.InputStateUpdatedEvent{-1, -1}
		ui.notifyInputStateUpdated(e)
		return
	}

	x, y := int(cx), int(cy)
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
	e := ebiten.InputStateUpdatedEvent{x, y}
	ui.notifyInputStateUpdated(e)
}

package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// #include "input.h"
//
// void StartApplication(void);
// void PollEvents(void);
// void BeginDrawing(void* window);
// void EndDrawing(void* window);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
	"unsafe"
)

type UI struct {
	screenWidth      int
	screenHeight     int
	screenScale      int
	graphicsDevice   *opengl.Device
	window           unsafe.Pointer
	initialEventSent bool
	textureFactory   *textureFactory
	uiEvents
}

var currentUI *UI

func New(screenWidth, screenHeight, screenScale int, title string) *UI {
	if currentUI != nil {
		panic("UI can't be duplicated.")
	}
	u := &UI{
		screenWidth:      screenWidth,
		screenHeight:     screenHeight,
		screenScale:      screenScale,
		initialEventSent: false,
	}

	C.StartApplication()

	u.textureFactory = runTextureFactory()

	u.textureFactory.UseContext(func() {
		u.graphicsDevice = opengl.NewDevice(
			u.screenWidth,
			u.screenHeight,
			u.screenScale)
	})

	u.window = u.textureFactory.CreateWindow(
		u.screenWidth*u.screenScale,
		u.screenHeight*u.screenScale,
		title)

	currentUI = u

	return u
}

func (u *UI) PollEvents() {
	C.PollEvents()
	if !u.initialEventSent {
		e := ui.ScreenSizeUpdatedEvent{u.screenWidth, u.screenHeight}
		u.uiEvents.notifyScreenSizeUpdated(e)
		u.initialEventSent = true
	}
}

func (u *UI) CreateRenderTarget(tag string, width, height int) {
}

func (u *UI) CreateTexture(tag string, img image.Image) {
}

func (u *UI) LoadResources(f func(graphics.TextureFactory)) {
	// This should be executed on the shared-context context
	f(u.graphicsDevice)
}

func (u *UI) Draw(f func(graphics.Canvas)) {
	C.BeginDrawing(u.window)
	u.graphicsDevice.Update(f)
	C.EndDrawing(u.window)
}

//export ebiten_ScreenSizeUpdated
func ebiten_ScreenSizeUpdated(width, height int) {
	u := currentUI
	e := ui.ScreenSizeUpdatedEvent{width, height}
	u.uiEvents.notifyScreenSizeUpdated(e)
}

//export ebiten_InputUpdated
func ebiten_InputUpdated(inputType C.InputType, cx, cy C.int) {
	u := currentUI

	if inputType == C.InputTypeMouseUp {
		e := ui.InputStateUpdatedEvent{-1, -1}
		u.uiEvents.notifyInputStateUpdated(e)
		return
	}

	x, y := int(cx), int(cy)
	x /= u.screenScale
	y /= u.screenScale
	if x < 0 {
		x = 0
	} else if u.screenWidth <= x {
		x = u.screenWidth - 1
	}
	if y < 0 {
		y = 0
	} else if u.screenHeight <= y {
		y = u.screenHeight - 1
	}
	e := ui.InputStateUpdatedEvent{x, y}
	u.uiEvents.notifyInputStateUpdated(e)
}

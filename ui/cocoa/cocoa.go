package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// #include "input.h"
//
// void StartApplication(void);
// void PollEvents(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
)

type UI struct {
	screenWidth      int
	screenHeight     int
	screenScale      int
	window           *Window
	initialEventSent bool
	textureFactory   *textureFactory
	graphicsDevice   *opengl.Device
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

	u.textureFactory.useContext(func() {
		u.graphicsDevice = opengl.NewDevice(
			u.screenWidth,
			u.screenHeight,
			u.screenScale)
	})
	u.window = u.CreateWindow(
		u.screenWidth,
		u.screenHeight,
		u.screenScale,
		title)

	currentUI = u

	return u
}

func (u *UI) CreateWindow(width, height, scale int, title string) *Window {
	return u.textureFactory.createWindow(u, width, height, scale, title)
}

func (u *UI) PollEvents() {
	C.PollEvents()
	if !u.initialEventSent {
		e := ui.ScreenSizeUpdatedEvent{u.screenWidth, u.screenHeight}
		u.uiEvents.notifyScreenSizeUpdated(e)
		u.initialEventSent = true
	}
}

func (u *UI) CreateTexture(tag interface{}, img image.Image) {
	go func() {
		var id graphics.TextureId
		var err error
		u.textureFactory.useContext(func() {
			id, err = u.graphicsDevice.CreateTexture(img)
		})
		e := graphics.TextureCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactory.notifyTextureCreated(e)
	}()
}

func (u *UI) CreateRenderTarget(tag interface{}, width, height int) {
	go func() {
		var id graphics.RenderTargetId
		var err error
		u.textureFactory.useContext(func() {
			id, err = u.graphicsDevice.CreateRenderTarget(width, height)
		})
		e := graphics.RenderTargetCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactory.notifyRenderTargetCreated(e)
	}()
}

func (u *UI) TextureCreated() <-chan graphics.TextureCreatedEvent {
	return u.textureFactory.TextureCreated()
}

func (u *UI) RenderTargetCreated() <-chan graphics.RenderTargetCreatedEvent {
	return u.textureFactory.RenderTargetCreated()
}

func (u *UI) Draw(f func(graphics.Canvas)) {
	u.window.Draw(f)
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

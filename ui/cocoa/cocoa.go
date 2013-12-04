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
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"github.com/hajimehoshi/go-ebiten/ui"
	"unsafe"
)

type UI struct {
	screenWidth       int
	screenHeight      int
	screenScale       int
	graphicsDevice    *opengl.Device
	window            unsafe.Pointer
	initialEventSent  bool
	screenSizeUpdated chan ui.ScreenSizeUpdatedEvent // initialized lazily
	inputStateUpdated chan ui.InputStateUpdatedEvent // initialized lazily
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

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	C.StartApplication()

	context := C.CreateGLContext(unsafe.Pointer(nil))
	C.SetCurrentGLContext(context)
	u.graphicsDevice = opengl.NewDevice(
		u.screenWidth,
		u.screenHeight,
		u.screenScale)

	u.window = C.CreateWindow(C.size_t(u.screenWidth*u.screenScale),
		C.size_t(u.screenHeight*u.screenScale),
		cTitle,
		context)
	currentUI = u

	return u
}

func (u *UI) PollEvents() {
	C.PollEvents()
	if !u.initialEventSent {
		e := ui.ScreenSizeUpdatedEvent{u.screenWidth, u.screenHeight}
		u.notifyScreenSizeUpdated(e)
		u.initialEventSent = true
	}
}

func (u *UI) LoadTextures(map[int]string) {
	// TODO: Implement
}

func (u *UI) LoadResources(f func(graphics.TextureFactory)) {
	C.BeginDrawing(u.window)
	f(u.graphicsDevice)
	C.EndDrawing(u.window)
}

func (u *UI) Draw(f func(graphics.Canvas)) {
	C.BeginDrawing(u.window)
	u.graphicsDevice.Update(f)
	C.EndDrawing(u.window)
}

func (u *UI) ScreenSizeUpdated() <-chan ui.ScreenSizeUpdatedEvent {
	if u.screenSizeUpdated != nil {
		return u.screenSizeUpdated
	}
	u.screenSizeUpdated = make(chan ui.ScreenSizeUpdatedEvent)
	return u.screenSizeUpdated
}

func (u *UI) notifyScreenSizeUpdated(e ui.ScreenSizeUpdatedEvent) {
	if u.screenSizeUpdated == nil {
		return
	}
	go func() {
		u.screenSizeUpdated <- e
	}()
}

func (u *UI) InputStateUpdated() <-chan ui.InputStateUpdatedEvent {
	if u.inputStateUpdated != nil {
		return u.inputStateUpdated
	}
	u.inputStateUpdated = make(chan ui.InputStateUpdatedEvent)
	return u.inputStateUpdated
}

func (u *UI) notifyInputStateUpdated(e ui.InputStateUpdatedEvent) {
	if u.inputStateUpdated == nil {
		return
	}
	go func() {
		u.inputStateUpdated <- e
	}()
}

//export ebiten_ScreenSizeUpdated
func ebiten_ScreenSizeUpdated(width, height int) {
	u := currentUI
	e := ui.ScreenSizeUpdatedEvent{width, height}
	u.notifyScreenSizeUpdated(e)
}

//export ebiten_InputUpdated
func ebiten_InputUpdated(inputType C.InputType, cx, cy C.int) {
	u := currentUI

	if inputType == C.InputTypeMouseUp {
		e := ui.InputStateUpdatedEvent{-1, -1}
		u.notifyInputStateUpdated(e)
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
	u.notifyInputStateUpdated(e)
}

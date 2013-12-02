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
	"runtime"
	"unsafe"
)

func init() {
	runtime.LockOSThread()
}

type UI struct {
	screenWidth               int
	screenHeight              int
	screenScale               int
	graphicsDevice            *opengl.Device
	window                    unsafe.Pointer
	initialEventSent          bool
	inputStateUpdatedChs      chan chan ui.InputStateUpdatedEvent
	inputStateUpdatedNotified chan ui.InputStateUpdatedEvent
	screenSizeUpdatedChs      chan chan ui.ScreenSizeUpdatedEvent
	screenSizeUpdatedNotified chan ui.ScreenSizeUpdatedEvent
}

var currentUI *UI

func New(screenWidth, screenHeight, screenScale int, title string) *UI {
	if currentUI != nil {
		panic("UI can't be duplicated.")
	}
	u := &UI{
		screenWidth:               screenWidth,
		screenHeight:              screenHeight,
		screenScale:               screenScale,
		initialEventSent:          false,
		inputStateUpdatedChs:      make(chan chan ui.InputStateUpdatedEvent),
		inputStateUpdatedNotified: make(chan ui.InputStateUpdatedEvent),
		screenSizeUpdatedChs:      make(chan chan ui.ScreenSizeUpdatedEvent),
		screenSizeUpdatedNotified: make(chan ui.ScreenSizeUpdatedEvent),
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

	u.eventLoop()

	return u
}

func (u *UI) eventLoop() {
	go func() {
		inputStateUpdated := []chan ui.InputStateUpdatedEvent{}
		for {
			select {
			case ch := <-u.inputStateUpdatedChs:
				inputStateUpdated = append(inputStateUpdated, ch)
			case e := <-u.inputStateUpdatedNotified:
				for _, ch := range inputStateUpdated {
					ch <- e
					close(ch)
				}
				inputStateUpdated = inputStateUpdated[0:0]
			}
		}
	}()

	go func() {
		screenSizeUpdated := []chan ui.ScreenSizeUpdatedEvent{}
		for {
			select {
			case ch := <-u.screenSizeUpdatedChs:
				screenSizeUpdated = append(screenSizeUpdated, ch)
			case e := <-u.screenSizeUpdatedNotified:
				for _, ch := range screenSizeUpdated {
					ch <- e
					close(ch)
				}
				screenSizeUpdated = []chan ui.ScreenSizeUpdatedEvent{}
			}
		}
	}()
}

func (u *UI) PollEvents() {
	C.PollEvents()
	if !u.initialEventSent {
		e := ui.ScreenSizeUpdatedEvent{u.screenWidth, u.screenHeight}
		u.notifyScreenSizeUpdated(e)
		u.initialEventSent = true
	}
}

func (u *UI) LoadResources(f func(graphics.TextureFactory)) {
	C.BeginDrawing(u.window)
	f(u.graphicsDevice.TextureFactory())
	C.EndDrawing(u.window)
}

func (u *UI) Draw(f func(graphics.Canvas)) {
	C.BeginDrawing(u.window)
	u.graphicsDevice.Update(f)
	C.EndDrawing(u.window)
}

func (u *UI) ObserveInputStateUpdated() <-chan ui.InputStateUpdatedEvent {
	ch := make(chan ui.InputStateUpdatedEvent)
	go func() {
		u.inputStateUpdatedChs <- ch
	}()
	return ch
}

func (u *UI) notifyInputStateUpdated(e ui.InputStateUpdatedEvent) {
	go func() {
		u.inputStateUpdatedNotified <- e
	}()
}

func (u *UI) ObserveScreenSizeUpdated() <-chan ui.ScreenSizeUpdatedEvent {
	ch := make(chan ui.ScreenSizeUpdatedEvent)
	go func() {
		u.screenSizeUpdatedChs <- ch
	}()
	return ch
}

func (u *UI) notifyScreenSizeUpdated(e ui.ScreenSizeUpdatedEvent) {
	go func() {
		u.screenSizeUpdatedNotified <- e
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

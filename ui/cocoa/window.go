package cocoa

// #include <stdlib.h>
//
// #include "input.h"
//
// void* CreateWindow(size_t width, size_t height, const char* title, void* glContext);
// void* CreateGLContext(void* sharedGLContext);
//
// void* GetGLContext(void* window);
// void UseGLContext(void* glContext);
// void UnuseGLContext(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"github.com/hajimehoshi/go-ebiten/ui"
	"runtime"
	"unsafe"
)

type Window struct {
	ui           *cocoaUI
	screenWidth  int
	screenHeight int
	screenScale  int
	closed       bool
	native       unsafe.Pointer
	context       *opengl.Context
	funcs        chan func()
	funcsDone    chan struct{}
	windowEvents
}

var windows = map[unsafe.Pointer]*Window{}

func runWindow(ui *cocoaUI, width, height, scale int, title string, sharedContext unsafe.Pointer) *Window {
	w := &Window{
		ui:           ui,
		screenWidth:  width,
		screenHeight: height,
		screenScale:  scale,
		closed:       false,
		funcs:        make(chan func()),
		funcsDone:    make(chan struct{}),
	}

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		glContext := C.CreateGLContext(sharedContext)
		w.native = C.CreateWindow(C.size_t(width*scale),
			C.size_t(height*scale),
			cTitle,
			glContext)
		windows[w.native] = w
		close(ch)
		w.loop()
	}()
	<-ch
	w.useGLContext(func() {
		w.context = ui.graphicsDevice.CreateContext(width, height, scale)
	})
	return w
}

func (w *Window) loop() {
	for {
		select {
		case f := <-w.funcs:
			glContext := C.GetGLContext(w.native)
			C.UseGLContext(glContext)
			f()
			C.UnuseGLContext()
			w.funcsDone <- struct{}{}
		}
	}
}

func (w *Window) Draw(f func(graphics.Context)) {
	if w.closed {
		return
	}
	w.useGLContext(func() {
		w.ui.graphicsDevice.Update(w.context, f)
	})
}

func (w *Window) useGLContext(f func()) {
	w.funcs <- f
	<-w.funcsDone
}

/*//export ebiten_ScreenSizeUpdated
func ebiten_ScreenSizeUpdated(nativeWindow unsafe.Pointer, width, height int) {
	u := currentUI
	e := ui.ScreenSizeUpdatedEvent{width, height}
	u.windowEvents.notifyScreenSizeUpdated(e)
}*/

//export ebiten_InputUpdated
func ebiten_InputUpdated(nativeWindow unsafe.Pointer, inputType C.InputType, cx, cy C.int) {
	w := windows[nativeWindow]

	if inputType == C.InputTypeMouseUp {
		e := ui.InputStateUpdatedEvent{-1, -1}
		w.notifyInputStateUpdated(e)
		return
	}

	x, y := int(cx), int(cy)
	x /= w.screenScale
	y /= w.screenScale
	if x < 0 {
		x = 0
	} else if w.screenWidth <= x {
		x = w.screenWidth - 1
	}
	if y < 0 {
		y = 0
	} else if w.screenHeight <= y {
		y = w.screenHeight - 1
	}
	e := ui.InputStateUpdatedEvent{x, y}
	w.notifyInputStateUpdated(e)
}

//export ebiten_WindowClosed
func ebiten_WindowClosed(nativeWindow unsafe.Pointer) {
	w := windows[nativeWindow]
	w.closed = true
	w.notifyWindowClosed(ui.WindowClosedEvent{})
	delete(windows, nativeWindow)
}

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
	pressedKeys  map[ui.Key]struct{}
	context      *opengl.Context
	funcs        chan func()
	funcsDone    chan struct{}
	windowEvents
}

var windows = map[unsafe.Pointer]*Window{}

func runWindow(cocoaUI *cocoaUI, width, height, scale int, title string, sharedContext unsafe.Pointer) *Window {
	w := &Window{
		ui:           cocoaUI,
		screenWidth:  width,
		screenHeight: height,
		screenScale:  scale,
		closed:       false,
		pressedKeys:  map[ui.Key]struct{}{},
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
		w.context = w.ui.graphicsDevice.CreateContext(width, height, scale)
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

var cocoaKeyCodeToKey = map[int]ui.Key{
	123: ui.KeyLeft,
	124: ui.KeyRight,
	125: ui.KeyUp,
	126: ui.KeyDown,
}

//export ebiten_KeyDown
func ebiten_KeyDown(nativeWindow unsafe.Pointer, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	w.pressedKeys[key] = struct{}{}
}

//export ebiten_KeyUp
func ebiten_KeyUp(nativeWindow unsafe.Pointer, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	delete(w.pressedKeys, key)
}

//export ebiten_MouseStateUpdated
func ebiten_MouseStateUpdated(nativeWindow unsafe.Pointer, inputType C.InputType, cx, cy C.int) {
	w := windows[nativeWindow]

	if inputType == C.InputTypeMouseUp {
		e := ui.MouseStateUpdatedEvent{-1, -1}
		w.notify(e)
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
	e := ui.MouseStateUpdatedEvent{x, y}
	w.notify(e)
}

//export ebiten_WindowClosed
func ebiten_WindowClosed(nativeWindow unsafe.Pointer) {
	w := windows[nativeWindow]
	w.closed = true
	w.notify(ui.WindowClosedEvent{})
	delete(windows, nativeWindow)
}

package cocoa

// #include <stdlib.h>
//
// #include "input.h"
//
// @class NSWindow;
// @class NSOpenGLContext;
//
// typedef NSWindow* NSWindowPtr;
//
// NSWindow* CreateWindow(size_t width, size_t height, const char* title, NSOpenGLContext* glContext);
// NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext);
//
// NSOpenGLContext* GetGLContext(NSWindow* window);
// void UseGLContext(NSOpenGLContext* glContext);
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
	native       *C.NSWindow
	pressedKeys  map[ui.Key]struct{}
	context      *opengl.Context
	funcs        chan func()
	funcsDone    chan struct{}
	events       chan interface{}
}

var windows = map[*C.NSWindow]*Window{}

func runWindow(cocoaUI *cocoaUI, width, height, scale int, title string, sharedContext *C.NSOpenGLContext) *Window {
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

func (w *Window) Events() <-chan interface{} {
	if w.events != nil {
		return w.events
	}
	w.events = make(chan interface{})
	return w.events
}

func (w *Window) notify(e interface{}) {
	if w.events == nil {
		return
	}
	go func() {
		w.events <- e
	}()
}

// Now this function is not used anywhere.
//export ebiten_WindowSizeUpdated
func ebiten_WindowSizeUpdated(nativeWindow C.NSWindowPtr, width, height int) {
	w := windows[nativeWindow]
	e := ui.WindowSizeUpdatedEvent{width, height}
	w.notify(e)
}

func (w *Window) keyStateUpdatedEvent() ui.KeyStateUpdatedEvent {
	keys := []ui.Key{}
	for key, _ := range w.pressedKeys {
		keys = append(keys, key)
	}
	return ui.KeyStateUpdatedEvent{
		Keys: keys,
	}
}

var cocoaKeyCodeToKey = map[int]ui.Key{
	49:  ui.KeySpace,
	123: ui.KeyLeft,
	124: ui.KeyRight,
	125: ui.KeyDown,
	126: ui.KeyUp,
}

//export ebiten_KeyDown
func ebiten_KeyDown(nativeWindow C.NSWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	w.pressedKeys[key] = struct{}{}
	w.notify(w.keyStateUpdatedEvent())
}

//export ebiten_KeyUp
func ebiten_KeyUp(nativeWindow C.NSWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	delete(w.pressedKeys, key)
	w.notify(w.keyStateUpdatedEvent())
}

//export ebiten_MouseStateUpdated
func ebiten_MouseStateUpdated(nativeWindow C.NSWindowPtr, inputType C.InputType, cx, cy C.int) {
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
func ebiten_WindowClosed(nativeWindow C.NSWindowPtr) {
	w := windows[nativeWindow]
	w.closed = true
	w.notify(ui.WindowClosedEvent{})
	delete(windows, nativeWindow)
}

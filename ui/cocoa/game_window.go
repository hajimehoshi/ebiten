package cocoa

// #include <stdlib.h>
//
// #include "input.h"
//
// @class EbitenGameWindow;
// @class NSOpenGLContext;
//
// typedef EbitenGameWindow* EbitenGameWindowPtr;
//
// EbitenGameWindow* CreateGameWindow(size_t width, size_t height, const char* title, NSOpenGLContext* glContext);
// NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext);
//
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

type GameWindow struct {
	screenWidth  int
	screenHeight int
	screenScale  int
	title        string
	native       *C.EbitenGameWindow
	pressedKeys  map[ui.Key]struct{}
	funcs        chan func(*opengl.Context)
	funcsDone    chan struct{}
	closed       chan struct{}
	events       chan interface{}
}

var windows = map[*C.EbitenGameWindow]*GameWindow{}

func newGameWindow(width, height, scale int, title string) *GameWindow {
	return &GameWindow{
		screenWidth:  width,
		screenHeight: height,
		screenScale:  scale,
		title:        title,
		pressedKeys:  map[ui.Key]struct{}{},
		funcs:        make(chan func(*opengl.Context)),
		funcsDone:    make(chan struct{}),
		closed:       make(chan struct{}),
	}
}

func (w *GameWindow) run(graphicsSharedContext *opengl.SharedContext, sharedGLContext *C.NSOpenGLContext) {
	cTitle := C.CString(w.title)
	defer C.free(unsafe.Pointer(cTitle))

	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		glContext := C.CreateGLContext(sharedGLContext)
		w.native = C.CreateGameWindow(C.size_t(w.screenWidth*w.screenScale),
			C.size_t(w.screenHeight*w.screenScale),
			cTitle,
			glContext)
		windows[w.native] = w
		close(ch)

		C.UseGLContext(glContext)
		context := graphicsSharedContext.CreateContext(
			w.screenWidth, w.screenHeight, w.screenScale)
		C.UnuseGLContext()

		defer func() {
			C.UseGLContext(glContext)
			context.Dispose()
			C.UnuseGLContext()
		}()

		w.loop(context, glContext)
	}()
	<-ch
}

func (w *GameWindow) loop(context *opengl.Context, glContext *C.NSOpenGLContext) {
	for {
		select {
		case <-w.closed:
			return
		case f := <-w.funcs:
			C.UseGLContext(glContext)
			f(context)
			C.UnuseGLContext()
			w.funcsDone <- struct{}{}
		}
	}
}

func (w *GameWindow) Draw(f func(graphics.Context)) {
	//after := time.After(time.Duration(int64(time.Second) / 120))
	//defer <-after

	select {
	case <-w.closed:
		return
	default:
	}
	w.useGLContext(func(context *opengl.Context) {
		context.Update(f)
	})
}

func (w *GameWindow) useGLContext(f func(*opengl.Context)) {
	w.funcs <- f
	<-w.funcsDone
}

func (w *GameWindow) Events() <-chan interface{} {
	if w.events != nil {
		return w.events
	}
	w.events = make(chan interface{})
	return w.events
}

func (w *GameWindow) notify(e interface{}) {
	if w.events == nil {
		return
	}
	go func() {
		w.events <- e
	}()
}

// Now this function is not used anywhere.
//export ebiten_WindowSizeUpdated
func ebiten_WindowSizeUpdated(nativeWindow C.EbitenGameWindowPtr, width, height int) {
	w := windows[nativeWindow]
	e := ui.WindowSizeUpdatedEvent{width, height}
	w.notify(e)
}

func (w *GameWindow) keyStateUpdatedEvent() ui.KeyStateUpdatedEvent {
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
func ebiten_KeyDown(nativeWindow C.EbitenGameWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	w.pressedKeys[key] = struct{}{}
	w.notify(w.keyStateUpdatedEvent())
}

//export ebiten_KeyUp
func ebiten_KeyUp(nativeWindow C.EbitenGameWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	delete(w.pressedKeys, key)
	w.notify(w.keyStateUpdatedEvent())
}

//export ebiten_MouseStateUpdated
func ebiten_MouseStateUpdated(nativeWindow C.EbitenGameWindowPtr, inputType C.InputType, cx, cy C.int) {
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
func ebiten_WindowClosed(nativeWindow C.EbitenGameWindowPtr) {
	w := windows[nativeWindow]
	close(w.closed)
	w.notify(ui.WindowClosedEvent{})
	delete(windows, nativeWindow)
}

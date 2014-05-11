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
	"sync"
	"time"
	"unsafe"
)

type GameWindow struct {
	state       ui.CanvasState
	title       string
	native      *C.EbitenGameWindow
	pressedKeys map[ui.Key]struct{}
	funcs       chan func(*opengl.Context)
	funcsDone   chan struct{}
	closed      chan struct{}
	sync.RWMutex
}

var windows = map[*C.EbitenGameWindow]*GameWindow{}

func newGameWindow(width, height, scale int, title string) *GameWindow {
	state := ui.CanvasState{
		Width:    width,
		Height:   height,
		Scale:    scale,
		Keys:     []ui.Key{},
		MouseX:   -1,
		MouseY:   -1,
		IsClosed: false,
	}
	return &GameWindow{
		state:       state,
		title:       title,
		pressedKeys: map[ui.Key]struct{}{},
		funcs:       make(chan func(*opengl.Context)),
		funcsDone:   make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (w *GameWindow) run(sharedGLContext *C.NSOpenGLContext) {
	cTitle := C.CString(w.title)
	defer C.free(unsafe.Pointer(cTitle))

	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		glContext := C.CreateGLContext(sharedGLContext)
		w.native = C.CreateGameWindow(
			C.size_t(w.state.Width*w.state.Scale),
			C.size_t(w.state.Height*w.state.Scale),
			cTitle,
			glContext)
		windows[w.native] = w
		close(ch)

		C.UseGLContext(glContext)
		context := opengl.NewContext(
			w.state.Width, w.state.Height, w.state.Scale)
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
			// Wait 10 millisecond at least to avoid busy loop.
			after := time.After(time.Duration(int64(time.Millisecond) * 10))
			C.UseGLContext(glContext)
			f(context)
			C.UnuseGLContext()
			<-after
			w.funcsDone <- struct{}{}
		}
	}
}

func (w *GameWindow) Draw(f func(graphics.Context)) {
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

func (w *GameWindow) State() ui.CanvasState {
	w.RLock()
	defer w.RUnlock()
	return w.state
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

	keys := []ui.Key{}
	for key, _ := range w.pressedKeys {
		keys = append(keys, key)
	}

	w.Lock()
	defer w.Unlock()
	w.state.Keys = keys
}

//export ebiten_KeyUp
func ebiten_KeyUp(nativeWindow C.EbitenGameWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]
	delete(w.pressedKeys, key)

	keys := []ui.Key{}
	for key, _ := range w.pressedKeys {
		keys = append(keys, key)
	}

	w.Lock()
	defer w.Unlock()
	w.state.Keys = keys
}

//export ebiten_MouseStateUpdated
func ebiten_MouseStateUpdated(nativeWindow C.EbitenGameWindowPtr, inputType C.InputType, cx, cy C.int) {
	w := windows[nativeWindow]

	if inputType == C.InputTypeMouseUp {
		w.Lock()
		defer w.Unlock()
		w.state.MouseX = -1
		w.state.MouseY = -1
		return
	}

	x, y := int(cx), int(cy)
	x /= w.state.Scale
	y /= w.state.Scale
	if x < 0 {
		x = 0
	} else if w.state.Width <= x {
		x = w.state.Width - 1
	}
	if y < 0 {
		y = 0
	} else if w.state.Height <= y {
		y = w.state.Height - 1
	}

	w.Lock()
	defer w.Unlock()
	w.state.MouseX = x
	w.state.MouseY = y
}

//export ebiten_WindowClosed
func ebiten_WindowClosed(nativeWindow C.EbitenGameWindowPtr) {
	w := windows[nativeWindow]
	close(w.closed)

	w.Lock()
	defer w.Unlock()
	w.state.IsClosed = true
	
	delete(windows, nativeWindow)
}

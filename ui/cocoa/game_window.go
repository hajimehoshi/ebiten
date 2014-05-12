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

type Keys map[ui.Key]struct{}

func newKeys() Keys {
	return Keys(map[ui.Key]struct{}{})
}

func (k Keys) clone() Keys {
	n := newKeys()
	for key, value := range k {
		n[key] = value
	}
	return n
}

func (k Keys) add(key ui.Key) {
	k[key] = struct{}{}
}

func (k Keys) remove(key ui.Key) {
	delete(k, key)
}

func (k Keys) Includes(key ui.Key) bool {
	_, ok := k[key]
	return ok
}

type InputState struct {
	pressedKeys Keys
	mouseX      int
	mouseY      int
}

func (i *InputState) PressedKeys() ui.Keys {
	return i.pressedKeys
}

func (i *InputState) MouseX() int {
	return i.mouseX
}

func (i *InputState) MouseY() int {
	return i.mouseY
}

func (i *InputState) setMouseXY(x, y int) {
	i.mouseX = x
	i.mouseY = y
}

type GameWindow struct {
	width      int
	height     int
	scale      int
	isClosed   bool
	inputState *InputState
	title      string
	native     *C.EbitenGameWindow
	funcs      chan func(*opengl.Context)
	funcsDone  chan struct{}
	closed     chan struct{}
	sync.RWMutex
}

var windows = map[*C.EbitenGameWindow]*GameWindow{}

func newGameWindow(width, height, scale int, title string) *GameWindow {
	inputState := &InputState{
		pressedKeys: newKeys(),
		mouseX:      -1,
		mouseY:      -1,
	}
	return &GameWindow{
		width:      width,
		height:     height,
		scale:      scale,
		inputState: inputState,
		title:      title,
		funcs:      make(chan func(*opengl.Context)),
		funcsDone:  make(chan struct{}),
		closed:     make(chan struct{}),
	}
}

func (w *GameWindow) IsClosed() bool {
	w.RLock()
	defer w.RUnlock()
	return w.isClosed
}

func (w *GameWindow) run(sharedGLContext *C.NSOpenGLContext) {
	cTitle := C.CString(w.title)
	defer C.free(unsafe.Pointer(cTitle))

	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		glContext := C.CreateGLContext(sharedGLContext)
		w.native = C.CreateGameWindow(
			C.size_t(w.width*w.scale),
			C.size_t(w.height*w.scale),
			cTitle,
			glContext)
		windows[w.native] = w
		close(ch)

		C.UseGLContext(glContext)
		context := opengl.NewContext(
			w.width, w.height, w.scale)
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

func (w *GameWindow) InputState() ui.InputState {
	w.RLock()
	defer w.RUnlock()
	return &InputState{
		pressedKeys: w.inputState.pressedKeys.clone(),
		mouseX:      w.inputState.mouseX,
		mouseY:      w.inputState.mouseY,
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

	w.Lock()
	defer w.Unlock()
	w.inputState.pressedKeys.add(key)
}

//export ebiten_KeyUp
func ebiten_KeyUp(nativeWindow C.EbitenGameWindowPtr, keyCode int) {
	key, ok := cocoaKeyCodeToKey[keyCode]
	if !ok {
		return
	}
	w := windows[nativeWindow]

	w.Lock()
	defer w.Unlock()
	w.inputState.pressedKeys.remove(key)
}

//export ebiten_MouseStateUpdated
func ebiten_MouseStateUpdated(nativeWindow C.EbitenGameWindowPtr, inputType C.InputType, cx, cy C.int) {
	w := windows[nativeWindow]

	if inputType == C.InputTypeMouseUp {
		w.Lock()
		defer w.Unlock()
		w.inputState.setMouseXY(-1, -1)
		return
	}

	x, y := int(cx), int(cy)
	x /= w.scale
	y /= w.scale
	if x < 0 {
		x = 0
	} else if w.width <= x {
		x = w.width - 1
	}
	if y < 0 {
		y = 0
	} else if w.height <= y {
		y = w.height - 1
	}

	w.Lock()
	defer w.Unlock()
	w.inputState.setMouseXY(x, y)
}

//export ebiten_WindowClosed
func ebiten_WindowClosed(nativeWindow C.EbitenGameWindowPtr) {
	w := windows[nativeWindow]
	close(w.closed)

	w.Lock()
	defer w.Unlock()
	w.isClosed = true

	delete(windows, nativeWindow)
}

package cocoa

// #include <stdlib.h>
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
	"runtime"
	"unsafe"
)

type Window struct {
	ui        *UI
	native    unsafe.Pointer
	canvas    *opengl.Canvas
	funcs     chan func()
	funcsDone chan struct{}
}

func runWindow(ui *UI, width, height, scale int, title string, sharedContext unsafe.Pointer) *Window {
	w := &Window{
		ui:        ui,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
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
		close(ch)
		w.loop()
	}()
	<-ch
	w.useContext(func() {
		w.canvas = ui.graphicsDevice.CreateCanvas(width, height, scale)
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

func (w *Window) Draw(f func(graphics.Canvas)) {
	w.useContext(func() {
		w.ui.graphicsDevice.Update(w.canvas, f)
	})
}

func (w *Window) useContext(f func()) {
	w.funcs <- f
	<-w.funcsDone
}

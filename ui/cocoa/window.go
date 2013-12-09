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
	"runtime"
	"unsafe"
)

type window struct {
	ui        *UI
	native    unsafe.Pointer
	funcs     chan func()
	funcsDone chan struct{}
}

func runWindow(ui *UI, width, height int, title string, sharedContext unsafe.Pointer) *window {
	w := &window{
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
		w.native = C.CreateWindow(C.size_t(width),
			C.size_t(height),
			cTitle,
			glContext)
		close(ch)
		w.loop()
	}()
	<-ch
	return w
}

func (w *window) loop() {
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

func (w *window) Draw(f func(graphics.Canvas)) {
	w.useContext(func() {
		w.ui.graphicsDevice.Update(f)
	})
}

func (w *window) useContext(f func()) {
	w.funcs <- f
	<-w.funcsDone
}

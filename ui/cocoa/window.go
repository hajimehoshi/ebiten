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
	"runtime"
	"unsafe"
)

type window struct {
	native    unsafe.Pointer
	funcs     chan func()
	funcsDone chan struct{}
}

func runWindow(width, height int, title string, sharedContext unsafe.Pointer) *window {
	w := &window{
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
	// TODO: Activate here?
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

func (w *window) UseContext(f func()) {
	w.funcs <- f
	<-w.funcsDone
}

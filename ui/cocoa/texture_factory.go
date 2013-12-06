package cocoa

// #include <stdlib.h>
//
// void* CreateGLContext(void* sharedGLContext);
// void* CreateWindow(size_t width, size_t height, const char* title, void* sharedGLContext);
// void UseGLContext(void* glContext);
//
import "C"
import (
	//"github.com/hajimehoshi/go-ebiten/graphics"
	"unsafe"
)

type textureFactory struct {
	sharedContext unsafe.Pointer
	funcs         chan func()
	funcsDone     chan struct{}
	textureFactoryEvents
}

func runTextureFactory() *textureFactory {
	t := &textureFactory{
		funcs: make(chan func()),
		funcsDone: make(chan struct{}),
	}
	ch := make(chan struct{})
	go func() {
		t.sharedContext = C.CreateGLContext(unsafe.Pointer(nil))
		close(ch)
		t.loop()
	}()
	<-ch
	return t
}

func (t *textureFactory) loop() {
	for {
		select {
		case f := <-t.funcs:
			C.UseGLContext(t.sharedContext)
			f()
			t.funcsDone <- struct{}{}
			// TODO: Unuse
		}
	}
}

func (t *textureFactory) UseContext(f func()) {
	t.funcs <- f
	<-t.funcsDone
}

func (t *textureFactory) CreateWindow(width, height int, title string) unsafe.Pointer {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	return C.CreateWindow(C.size_t(width),
		C.size_t(height),
		cTitle,
		t.sharedContext)
}

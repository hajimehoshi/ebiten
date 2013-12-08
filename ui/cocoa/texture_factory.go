package cocoa

// void* CreateGLContext(void* sharedGLContext);
// void UseGLContext(void* glContext);
// void UnuseGLContext(void);
//
import "C"
import (
	"runtime"
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
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}
	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
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
			C.UnuseGLContext()
			t.funcsDone <- struct{}{}
		}
	}
}

func (t *textureFactory) UseContext(f func()) {
	t.funcs <- f
	<-t.funcsDone
}

func (t *textureFactory) CreateWindow(width, height int, title string) *window {
	return runWindow(width, height, title, t.sharedContext)
}

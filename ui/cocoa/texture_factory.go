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

func (t *textureFactory) useGLContext(f func()) {
	t.funcs <- f
	<-t.funcsDone
}

func (t *textureFactory) createWindow(ui *cocoaUI, width, height, scale int, title string) *Window {
	return runWindow(ui, width, height, scale, title, t.sharedContext)
}

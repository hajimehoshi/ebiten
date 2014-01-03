package cocoa

// @class NSOpenGLContext;
//
// NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext);
// void UseGLContext(NSOpenGLContext* glContext);
// void UnuseGLContext(void);
//
import "C"
import (
	"runtime"
)

type textureFactory struct {
	sharedContext *C.NSOpenGLContext
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
		t.sharedContext = C.CreateGLContext(nil)
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

func (t *textureFactory) createGameWindow(ui *cocoaUI, width, height, scale int, title string) *GameWindow {
	return runGameWindow(ui, width, height, scale, title, t.sharedContext)
}

package cocoa

// @class NSOpenGLContext;
//
// NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext);
// void UseGLContext(NSOpenGLContext* glContext);
// void UnuseGLContext(void);
//
import "C"
import (
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/opengl"
	"image"
	"runtime"
)

type sharedContext struct {
	inited      chan struct{}
	funcs       chan func()
	funcsDone   chan struct{}
	gameWindows chan *GameWindow
}

func newSharedContext() *sharedContext {
	return &sharedContext{
		inited:      make(chan struct{}),
		funcs:       make(chan func()),
		funcsDone:   make(chan struct{}),
		gameWindows: make(chan *GameWindow),
	}
}

func (t *sharedContext) run() {
	var sharedGLContext *C.NSOpenGLContext
	go func() {
		runtime.LockOSThread()
		sharedGLContext = C.CreateGLContext(nil)
		close(t.inited)
		t.loop(sharedGLContext)
	}()
	<-t.inited
	go func() {
		for w := range t.gameWindows {
			w.run(sharedGLContext)
		}
	}()
}

func (t *sharedContext) loop(sharedGLContext *C.NSOpenGLContext) {
	for {
		select {
		case f := <-t.funcs:
			C.UseGLContext(sharedGLContext)
			f()
			C.UnuseGLContext()
			t.funcsDone <- struct{}{}
		}
	}
}

func (t *sharedContext) useGLContext(f func()) {
	t.funcs <- f
	<-t.funcsDone
}

func (t *sharedContext) createGameWindow(width, height, scale int, title string) *GameWindow {
	w := newGameWindow(width, height, scale, title)
	go func() {
		t.gameWindows <- w
	}()
	return w
}

func (t *sharedContext) CreateTexture(
	img image.Image,
	filter graphics.Filter) (graphics.TextureId, error) {
	<-t.inited
	var id graphics.TextureId
	var err error
	t.useGLContext(func() {
		id, err = opengl.CreateTexture(img, filter)
	})
	return id, err
}

func (t *sharedContext) CreateRenderTarget(
	width, height int,
	filter graphics.Filter) (graphics.RenderTargetId, error) {
	<-t.inited
	var id graphics.RenderTargetId
	var err error
	t.useGLContext(func() {
		id, err = opengl.CreateRenderTarget(width, height, filter)
	})
	return id, err
}

package cocoa

// @class NSOpenGLContext;
//
// NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext);
// void UseGLContext(NSOpenGLContext* glContext);
// void UnuseGLContext(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"image"
	"runtime"
)

type sharedContext struct {
	inited         chan struct{}
	graphicsSharedContext *opengl.SharedContext
	events         chan interface{}
	funcs          chan func()
	funcsDone      chan struct{}
	gameWindows    chan *GameWindow
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
		t.graphicsSharedContext = opengl.NewSharedContext()
		sharedGLContext = C.CreateGLContext(nil)
		close(t.inited)
		t.loop(sharedGLContext)
	}()
	<-t.inited
	go func() {
		for w := range t.gameWindows {
			w.run(t.graphicsSharedContext, sharedGLContext)
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

func (t *sharedContext) Events() <-chan interface{} {
	if t.events != nil {
		return t.events
	}
	t.events = make(chan interface{})
	return t.events
}

func (t *sharedContext) CreateTexture(tag interface{}, img image.Image, filter graphics.Filter) {
	go func() {
		<-t.inited
		var id graphics.TextureId
		var err error
		t.useGLContext(func() {
			id, err = t.graphicsSharedContext.CreateTexture(img, filter)
		})
		if t.events == nil {
			return
		}
		t.events <- graphics.TextureCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
	}()
}

func (t *sharedContext) CreateRenderTarget(tag interface{}, width, height int) {
	go func() {
		<-t.inited
		var id graphics.RenderTargetId
		var err error
		t.useGLContext(func() {
			id, err = t.graphicsSharedContext.CreateRenderTarget(width, height)
		})
		if t.events == nil {
			return
		}
		t.events <- graphics.RenderTargetCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
	}()
}

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

type textureFactory struct {
	inited         chan struct{}
	graphicsDevice *opengl.Device
	events         chan interface{}
	funcs          chan func()
	funcsDone      chan struct{}
	gameWindows    chan *GameWindow
}

func newTextureFactory() *textureFactory {
	return &textureFactory{
		inited:      make(chan struct{}),
		funcs:       make(chan func()),
		funcsDone:   make(chan struct{}),
		gameWindows: make(chan *GameWindow),
	}
}

func (t *textureFactory) run() {
	var sharedContext *C.NSOpenGLContext
	go func() {
		runtime.LockOSThread()
		t.graphicsDevice = opengl.NewDevice()
		sharedContext = C.CreateGLContext(nil)
		close(t.inited)
		t.loop(sharedContext)
	}()
	<-t.inited
	go func() {
		for w := range t.gameWindows {
			w.run(t.graphicsDevice, sharedContext)
		}
	}()
}

func (t *textureFactory) loop(sharedContext *C.NSOpenGLContext) {
	for {
		select {
		case f := <-t.funcs:
			C.UseGLContext(sharedContext)
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

func (t *textureFactory) createGameWindow(width, height, scale int, title string) *GameWindow {
	w := newGameWindow(width, height, scale, title)
	go func() {
		t.gameWindows <- w
	}()
	return w
}

func (t *textureFactory) Events() <-chan interface{} {
	if t.events != nil {
		return t.events
	}
	t.events = make(chan interface{})
	return t.events
}

func (t *textureFactory) CreateTexture(tag interface{}, img image.Image, filter graphics.Filter) {
	go func() {
		<-t.inited
		var id graphics.TextureId
		var err error
		t.useGLContext(func() {
			id, err = t.graphicsDevice.CreateTexture(img, filter)
		})
		if t.events == nil {
			return
		}
		e := graphics.TextureCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		t.events <- e
	}()
}

func (t *textureFactory) CreateRenderTarget(tag interface{}, width, height int) {
	go func() {
		<-t.inited
		var id graphics.RenderTargetId
		var err error
		t.useGLContext(func() {
			id, err = t.graphicsDevice.CreateRenderTarget(width, height)
		})
		if t.events == nil {
			return
		}
		e := graphics.RenderTargetCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		t.events <- e
	}()
}

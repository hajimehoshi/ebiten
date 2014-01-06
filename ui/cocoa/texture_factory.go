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
	sharedContext  *C.NSOpenGLContext
	graphicsDevice *opengl.Device
	events         chan interface{}
	funcs          chan func()
	funcsDone      chan struct{}
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
	t.useGLContext(func() {
		t.graphicsDevice = opengl.NewDevice()
	})
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

func (t *textureFactory) createGameWindow(width, height, scale int, title string) *GameWindow {
	return runGameWindow(t.graphicsDevice, width, height, scale, title, t.sharedContext)
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

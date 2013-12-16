package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// void StartApplication(void);
// void PollEvents(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
)

type cocoaUI struct {
	textureFactory *textureFactory
	textureFactoryEvents chan interface{}
	graphicsDevice *opengl.Device
}

var currentUI *cocoaUI

func getCurrentUI() *cocoaUI {
	if currentUI != nil {
		return currentUI
	}

	currentUI = &cocoaUI{}

	C.StartApplication()

	currentUI.textureFactory = runTextureFactory()
	currentUI.textureFactory.useGLContext(func() {
		currentUI.graphicsDevice = opengl.NewDevice()
	})

	return currentUI
}

func UI() ui.UI {
	return getCurrentUI()
}

func TextureFactory() graphics.TextureFactory {
	return getCurrentUI()
}

func (u *cocoaUI) CreateWindow(width, height, scale int, title string) ui.Window {
	return u.textureFactory.createWindow(u, width, height, scale, title)
}

func (u *cocoaUI) PollEvents() {
	C.PollEvents()
}

func (u *cocoaUI) Events() <-chan interface{} {
	if u.textureFactoryEvents != nil {
		return u.textureFactoryEvents
	}
	u.textureFactoryEvents = make(chan interface{})
	return u.textureFactoryEvents
}

func (u *cocoaUI) CreateTexture(tag interface{}, img image.Image) {
	go func() {
		var id graphics.TextureId
		var err error
		u.textureFactory.useGLContext(func() {
			id, err = u.graphicsDevice.CreateTexture(img)
		})
		e := graphics.TextureCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactoryEvents <- e
	}()
}

func (u *cocoaUI) CreateRenderTarget(tag interface{}, width, height int) {
	go func() {
		var id graphics.RenderTargetId
		var err error
		u.textureFactory.useGLContext(func() {
			id, err = u.graphicsDevice.CreateRenderTarget(width, height)
		})
		e := graphics.RenderTargetCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactoryEvents <- e
	}()
}

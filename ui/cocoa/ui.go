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

type UI struct {
	textureFactory *textureFactory
	graphicsDevice *opengl.Device
}

var currentUI *UI

func NewUI() *UI {
	if currentUI != nil {
		panic("UI can't be duplicated.")
	}
	u := &UI{}

	C.StartApplication()

	u.textureFactory = runTextureFactory()
	u.textureFactory.useContext(func() {
		u.graphicsDevice = opengl.NewDevice()
	})

	currentUI = u

	return u
}

func (u *UI) CreateWindow(width, height, scale int, title string) ui.Window {
	return u.textureFactory.createWindow(u, width, height, scale, title)
}

func (u *UI) PollEvents() {
	C.PollEvents()
}

func (u *UI) CreateTexture(tag interface{}, img image.Image) {
	go func() {
		var id graphics.TextureId
		var err error
		u.textureFactory.useContext(func() {
			id, err = u.graphicsDevice.CreateTexture(img)
		})
		e := graphics.TextureCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactory.notifyTextureCreated(e)
	}()
}

func (u *UI) CreateRenderTarget(tag interface{}, width, height int) {
	go func() {
		var id graphics.RenderTargetId
		var err error
		u.textureFactory.useContext(func() {
			id, err = u.graphicsDevice.CreateRenderTarget(width, height)
		})
		e := graphics.RenderTargetCreatedEvent{
			Tag:   tag,
			Id:    id,
			Error: err,
		}
		u.textureFactory.notifyRenderTargetCreated(e)
	}()
}

func (u *UI) TextureCreated() <-chan graphics.TextureCreatedEvent {
	return u.textureFactory.TextureCreated()
}

func (u *UI) RenderTargetCreated() <-chan graphics.RenderTargetCreatedEvent {
	return u.textureFactory.RenderTargetCreated()
}

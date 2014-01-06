package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// void Run(void);
// void StartApplication(void);
// void PollEvents(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
)

type cocoaUI struct {
	textureFactory *textureFactory
}

var currentUI *cocoaUI

func getCurrentUI() *cocoaUI {
	if currentUI != nil {
		return currentUI
	}

	currentUI = &cocoaUI{}

	C.StartApplication()

	currentUI.textureFactory = runTextureFactory()
	return currentUI
}

func UI() ui.UI {
	return getCurrentUI()
}

func TextureFactory() graphics.TextureFactory {
	return getCurrentUI().textureFactory
}

func (u *cocoaUI) CreateGameWindow(width, height, scale int, title string) ui.GameWindow {
	return u.textureFactory.createGameWindow(width, height, scale, title)
}

func (u *cocoaUI) PollEvents() {
	C.PollEvents()
}

func (u *cocoaUI) MainLoop() {
	C.Run()
}

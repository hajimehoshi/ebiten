package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// void Run(void);
// void StartApplication(void);
// void DoEvents(void);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
)

type cocoaUI struct {
	sharedContext *sharedContext
}

var currentUI *cocoaUI

func getCurrentUI() *cocoaUI {
	if currentUI != nil {
		return currentUI
	}

	currentUI = &cocoaUI{}
	currentUI.sharedContext = newSharedContext()

	return currentUI
}

func UI() ui.UI {
	return getCurrentUI()
}

func TextureFactory() graphics.TextureFactory {
	return getCurrentUI().sharedContext
}

func (u *cocoaUI) CreateGameWindow(width, height, scale int, title string) ui.GameWindow {
	return u.sharedContext.createGameWindow(width, height, scale, title)
}

func (u *cocoaUI) DoEvents() {
	C.DoEvents()
}

func (u *cocoaUI) RunMainLoop() {
	C.StartApplication()
	currentUI.sharedContext.run()

	// TODO: Enable the loop
	//C.Run()
}

/*func (u *cocoaUI) CreateTexture(tag interface{}, img image.Image, filter graphics.Filter) {
	t.sharedContext.CreateTexture(tag, img, filter)
}

func (u *cocoaUI) CreateRenderTarget(tag interface{}, width, height int) {
	t.sharedContext.CreateRenderTarget(tag, width, height)
}*/

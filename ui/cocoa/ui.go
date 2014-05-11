package cocoa

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL
//
// void StartApplication(void);
// void DoEvents(void);
// void TerminateApplication(void);
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

func (u *cocoaUI) CreateCanvas(width, height, scale int, title string) ui.Canvas {
	return u.sharedContext.createGameWindow(width, height, scale, title)
}

func (u *cocoaUI) DoEvents() {
	C.DoEvents()
}

func (u *cocoaUI) Start() {
	C.StartApplication()
	currentUI.sharedContext.run()
}

func (u *cocoaUI) Terminate() {
	// TODO: Close existing windows
	C.TerminateApplication()
}

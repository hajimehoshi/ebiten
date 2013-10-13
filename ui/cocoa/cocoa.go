package cocoa

// #cgo CFLAGS: -x objective-c -fobjc-arc
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework QuartzCore
//
// #include <stdlib.h>
//
// void Run(size_t width, size_t height, size_t scale, const char* title);
//
import "C"
import (
	"github.com/hajimehoshi/go.ebiten"
	_ "github.com/hajimehoshi/go.ebiten/graphics"
	_ "github.com/hajimehoshi/go.ebiten/graphics/opengl"
	"unsafe"
)

func Run(game ebiten.Game, screenWidth, screenHeight, screenScale int,
	title string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	C.Run(C.size_t(screenWidth), C.size_t(screenHeight), C.size_t(screenScale), cTitle)
}

package cocoa

// #cgo CFLAGS: -x objective-c -fobjc-arc
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework QuartzCore
//
// #include <stdlib.h>
//
// void Run(size_t width, size_t height, size_t scale);
//
import "C"
import (
	"github.com/hajimehoshi/go.ebiten"
	_ "github.com/hajimehoshi/go.ebiten/graphics"
	_ "github.com/hajimehoshi/go.ebiten/graphics/opengl"
)

func Run(game ebiten.Game, screenWidth, screenHeight, screenScale int,
	title string) {
	C.Run(C.size_t(screenWidth), C.size_t(screenHeight), C.size_t(screenScale))
}

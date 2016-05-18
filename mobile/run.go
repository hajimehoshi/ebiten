package mobile

import (
	"github.com/hajimehoshi/ebiten"
)

var chError <-chan error

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height, scale int, title string) {
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, title)
	return
}

func LastErrorString() string {
	select {
	case err := <-chError:
		return err.Error()
	default:
		return ""
	}
}

func SetScreenSize(width, height int) {
	// TODO: Implement this
}

func SetScreenScale(scale int) {
	// TODO: Implement this
}

func Render() {
	// TODO: Implement this
}

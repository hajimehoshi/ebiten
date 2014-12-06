package ui

type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyMax
)

var currentKeyboard Keyboard

type Keyboard interface {
	IsKeyPressed(key Key) bool
}

func SetKeyboard(keyboard Keyboard) {
	currentKeyboard = keyboard
}

func IsKeyPressed(key Key) bool {
	if currentKeyboard == nil {
		panic("ui.IsKeyPressed: currentKeyboard is not set")
	}
	return currentKeyboard.IsKeyPressed(key)
}

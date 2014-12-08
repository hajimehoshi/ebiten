package input

type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	MouseButtonMax
)

var currentMouse Mouse

type Mouse interface {
	CurrentPosition() (x, y int)
	IsMouseButtonPressed(mouseButton MouseButton) bool
}

func SetMouse(mouse Mouse) {
	currentMouse = mouse
}

func CurrentPosition() (x, y int) {
	if currentMouse == nil {
		panic("input.CurrentPosition: currentMouse is not set")
	}
	return currentMouse.CurrentPosition()
}

func IsMouseButtonPressed(button MouseButton) bool {
	if currentMouse == nil {
		panic("input.IsMouseButtonPressed: currentMouse is not set")
	}
	return currentMouse.IsMouseButtonPressed(button)
}

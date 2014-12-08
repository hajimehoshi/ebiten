package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/input"
	"math"
)

type mouse struct {
	buttonPressed [input.MouseButtonMax]bool
	x             int
	y             int
}

func (m *mouse) CurrentPosition() (x, y int) {
	return m.x, m.y
}

func (m *mouse) IsMouseButtonPressed(button input.MouseButton) bool {
	return m.buttonPressed[button]
}

func (m *mouse) update(window *glfw.Window) {
	x, y := window.GetCursorPosition()
	m.x = int(math.Floor(x))
	m.y = int(math.Floor(y))
	for i := input.MouseButtonLeft; i < input.MouseButtonMax; i++ {
		m.buttonPressed[i] = window.GetMouseButton(glfw.MouseButton(i)) == glfw.Press
	}
}

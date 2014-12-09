package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
	"math"
)

type mouse struct {
	buttonPressed [ebiten.MouseButtonMax]bool
	x             int
	y             int
}

func (m *mouse) CursorPosition() (x, y int) {
	return m.x, m.y
}

func (m *mouse) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	return m.buttonPressed[button]
}

func (m *mouse) update(window *glfw.Window) {
	x, y := window.GetCursorPosition()
	m.x = int(math.Floor(x))
	m.y = int(math.Floor(y))
	for i := ebiten.MouseButtonLeft; i < ebiten.MouseButtonMax; i++ {
		m.buttonPressed[i] = window.GetMouseButton(glfw.MouseButton(i)) == glfw.Press
	}
}

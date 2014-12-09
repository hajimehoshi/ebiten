package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
)

type keyboard struct {
	keyPressed [ebiten.KeyMax]bool
}

func (k *keyboard) IsKeyPressed(key ebiten.Key) bool {
	return k.keyPressed[key]
}

var glfwKeyCodeToKey = map[glfw.Key]ebiten.Key{
	glfw.KeySpace: ebiten.KeySpace,
	glfw.KeyLeft:  ebiten.KeyLeft,
	glfw.KeyRight: ebiten.KeyRight,
	glfw.KeyUp:    ebiten.KeyUp,
	glfw.KeyDown:  ebiten.KeyDown,
}

func (k *keyboard) update(window *glfw.Window) {
	for g, u := range glfwKeyCodeToKey {
		k.keyPressed[u] = window.GetKey(g) == glfw.Press
	}
}

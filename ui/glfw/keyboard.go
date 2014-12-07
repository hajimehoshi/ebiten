package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/input"
)

type keyboard struct {
	keyPressed [input.KeyMax]bool
}

func (k *keyboard) IsKeyPressed(key input.Key) bool {
	return k.keyPressed[key]
}

var glfwKeyCodeToKey = map[glfw.Key]input.Key{
	glfw.KeySpace: input.KeySpace,
	glfw.KeyLeft:  input.KeyLeft,
	glfw.KeyRight: input.KeyRight,
	glfw.KeyUp:    input.KeyUp,
	glfw.KeyDown:  input.KeyDown,
}

func (k *keyboard) update(window *glfw.Window) {
	for g, u := range glfwKeyCodeToKey {
		k.keyPressed[u] = window.GetKey(g) == glfw.Press
	}
}

package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/ui"
)

type Keyboard struct {
	pressedKeys map[ui.Key]struct{}
}

func NewKeyboard() *Keyboard {
	return &Keyboard{
		pressedKeys: map[ui.Key]struct{}{},
	}
}

func (k *Keyboard) IsKeyPressed(key ui.Key) bool {
	_, ok := k.pressedKeys[key]
	return ok
}

var glfwKeyCodeToKey = map[glfw.Key]ui.Key{
	glfw.KeySpace: ui.KeySpace,
	glfw.KeyLeft:  ui.KeyLeft,
	glfw.KeyRight: ui.KeyRight,
	glfw.KeyUp:    ui.KeyUp,
	glfw.KeyDown:  ui.KeyDown,
}

func (k *Keyboard) update(window *glfw.Window) {
	for g, u := range glfwKeyCodeToKey {
		if window.GetKey(g) == glfw.Press {
			k.pressedKeys[u] = struct{}{}
		} else {
			delete(k.pressedKeys, u)
		}
	}
}

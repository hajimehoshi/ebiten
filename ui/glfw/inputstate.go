package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/ui"
)

type Keys map[ui.Key]struct{}

func newKeys() Keys {
	return Keys(map[ui.Key]struct{}{})
}

func (k Keys) add(key ui.Key) {
	k[key] = struct{}{}
}

func (k Keys) remove(key ui.Key) {
	delete(k, key)
}

func (k Keys) Includes(key ui.Key) bool {
	_, ok := k[key]
	return ok
}

type InputState struct {
	pressedKeys Keys
	mouseX      int
	mouseY      int
}

func newInputState() *InputState {
	return &InputState{
		pressedKeys: newKeys(),
		mouseX:      -1,
		mouseY:      -1,
	}
}

func (i *InputState) IsPressedKey(key ui.Key) bool {
	return i.pressedKeys.Includes(key)
}

func (i *InputState) MouseX() int {
	// TODO: Update
	return i.mouseX
}

func (i *InputState) MouseY() int {
	return i.mouseY
}

var glfwKeyCodeToKey = map[glfw.Key]ui.Key{
	glfw.KeySpace: ui.KeySpace,
	glfw.KeyLeft:  ui.KeyLeft,
	glfw.KeyRight: ui.KeyRight,
	glfw.KeyUp:    ui.KeyUp,
	glfw.KeyDown:  ui.KeyDown,
}

func (i *InputState) update(window *glfw.Window) {
	for g, u := range glfwKeyCodeToKey {
		if window.GetKey(g) == glfw.Press {
			i.pressedKeys.add(u)
		} else {
			i.pressedKeys.remove(u)
		}
	}
}

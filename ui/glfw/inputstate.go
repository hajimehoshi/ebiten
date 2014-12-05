package glfw

import (
	"github.com/hajimehoshi/ebiten/ui"
)

type Keys map[ui.Key]struct{}

func newKeys() Keys {
	return Keys(map[ui.Key]struct{}{})
}

func (k Keys) clone() Keys {
	n := newKeys()
	for key, value := range k {
		n[key] = value
	}
	return n
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

func (i *InputState) PressedKeys() ui.Keys {
	return i.pressedKeys
}

func (i *InputState) MouseX() int {
	return i.mouseX
}

func (i *InputState) MouseY() int {
	return i.mouseY
}

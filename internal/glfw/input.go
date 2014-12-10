/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
	"math"
)

type input struct {
	keyPressed         [ebiten.KeyMax]bool
	mouseButtonPressed [ebiten.MouseButtonMax]bool
	cursorX            int
	cursorY            int
}

func (i *input) IsKeyPressed(key ebiten.Key) bool {
	return i.keyPressed[key]
}

func (i *input) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	return i.mouseButtonPressed[button]
}

func (i *input) CursorPosition() (x, y int) {
	return i.cursorX, i.cursorY
}

var glfwKeyCodeToKey = map[glfw.Key]ebiten.Key{
	glfw.KeySpace: ebiten.KeySpace,
	glfw.KeyLeft:  ebiten.KeyLeft,
	glfw.KeyRight: ebiten.KeyRight,
	glfw.KeyUp:    ebiten.KeyUp,
	glfw.KeyDown:  ebiten.KeyDown,
}

func (i *input) update(window *glfw.Window, scale int) {
	for g, u := range glfwKeyCodeToKey {
		i.keyPressed[u] = window.GetKey(g) == glfw.Press
	}
	for b := ebiten.MouseButtonLeft; b < ebiten.MouseButtonMax; b++ {
		i.mouseButtonPressed[b] = window.GetMouseButton(glfw.MouseButton(b)) == glfw.Press
	}
	x, y := window.GetCursorPosition()
	i.cursorX = int(math.Floor(x)) / scale
	i.cursorY = int(math.Floor(y)) / scale
}

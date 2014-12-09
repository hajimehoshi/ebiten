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

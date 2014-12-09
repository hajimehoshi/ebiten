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

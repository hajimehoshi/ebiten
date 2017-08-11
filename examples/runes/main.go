// +build example

// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"io"
)

var runes = append(make([]rune, 0, 1024), []rune("Type on your keyboard, Control-D to exit:\n")...)

var buf = make([]rune, 1024)

var counter int

func update(screen *ebiten.Image) error {
	n := ebiten.ReadInput(buf)
	runes = append(runes, buf[:n]...)
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if len(runes) > 0 && runes[len(runes)-1] != '\n' {
			runes = append(runes, '\n')
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyD) {
		return io.EOF
	}
	counter++
	switch counter%60 < 30 {
	case true:
		return ebitenutil.DebugPrint(screen, string(append(runes, '_')))
	}
	return ebitenutil.DebugPrint(screen, string(runes))
}

func main() {
	ebiten.Run(update, 320, 240, 2.0, "Runes (Ebiten Demo)") // ebiterm?
}

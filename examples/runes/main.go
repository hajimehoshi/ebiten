// +build example

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var runes = []rune("Type some stuff on your keyboard:\n")

func update(screen *ebiten.Image) error {
	runes = append(runes, ebiten.Keyboard()...)
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if len(runes) > 0 && runes[len(runes)-1] != '\n' {
			runes = append(runes, '\n')
		}
	}
	return ebitenutil.DebugPrint(screen, string(runes))
}

func main() {
	ebiten.Run(update, 320, 240, 2.0, "Hi")
}

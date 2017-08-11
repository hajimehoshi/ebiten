// +build example

package main

import (
	"io"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var runes = append(make([]rune,0,1024),[]rune("Type on your keyboard, Control-D to exit:\n")...)

var counter int

func update(screen *ebiten.Image) error {
	runes = append(runes, ebiten.InputChars()...)
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
		return ebitenutil.DebugPrint(screen, string(append(runes,'_')))
	}
	return ebitenutil.DebugPrint(screen, string(runes))
}

func main() {
	ebiten.Run(update, 320, 240, 2.0, "Runes (Ebiten Demo)") // ebiterm?
}

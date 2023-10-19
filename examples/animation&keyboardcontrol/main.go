package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"runningman/game"
)

const (
	screenWidth  = 1000
	screenHeight = 800
)

func main() {
	g := game.NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("runningman")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

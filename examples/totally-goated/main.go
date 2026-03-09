package main

import (
	"embed"
	"log"
	"runtime"
	"totally-goated/game"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*
var assets embed.FS

func main() {
	game.InitAssets(assets)
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("Totally Goated")
	if runtime.GOOS != "js" {
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.RunGame(game.NewGame()); err != nil {
		log.Fatal(err)
	}
}

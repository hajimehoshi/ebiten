package main

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/example/game/blank"
	"github.com/hajimehoshi/go-ebiten/example/game/input"
	"github.com/hajimehoshi/go-ebiten/example/game/monochrome"
	"github.com/hajimehoshi/go-ebiten/example/game/rects"
	"github.com/hajimehoshi/go-ebiten/example/game/rotating"
	"github.com/hajimehoshi/go-ebiten/example/game/sprites"
	"github.com/hajimehoshi/go-ebiten/example/game/testpattern"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	gameName := ""
	if 2 <= len(os.Args) {
		gameName = os.Args[1]
	}

	var game ebiten.Game
	switch gameName {
	case "blank":
		game = blank.New()
	case "input":
		game = input.New()
	case "monochrome":
		game = monochrome.New()
	case "rects":
		game = rects.New()
	default:
		fallthrough
	case "rotating":
		game = rotating.New()
	case "sprites":
		game = sprites.New()
	case "testpattern":
		game = testpattern.New()
	}

	const screenWidth = 256
	const screenHeight = 240
	const screenScale = 2
	const title = "Ebiten Demo"
	ui := cocoa.New(screenWidth, screenHeight, screenScale, title)
	ui.Start(game)
	ui.InitTextures(game)
	for {
		ui.WaitEvents()
		ui.Draw(game.Draw)
	}
}

package main

import (
	"github.com/hajimehoshi/ebiten/example/blocks"
	"github.com/hajimehoshi/ebiten/ui"
	"github.com/hajimehoshi/ebiten/ui/glfw"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	u := new(glfw.UI)
	game := blocks.NewGame()
	ui.Run(u, game, blocks.ScreenWidth, blocks.ScreenHeight, 2, "Ebiten Demo", 60)
}

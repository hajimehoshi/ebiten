package main

import (
	"github.com/hajimehoshi/ebiten/example/blocks"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/ui"
	"github.com/hajimehoshi/ebiten/ui/glfw"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Game interface {
	Update(state ui.InputState)
	Draw(c graphics.Context)
}

func init() {
	runtime.LockOSThread()
}

func main() {
	const screenWidth = blocks.ScreenWidth
	const screenHeight = blocks.ScreenHeight
	const screenScale = 2
	const fps = 60
	const frameTime = time.Duration(int64(time.Second) / int64(fps))
	const title = "Ebiten Demo"

	u := new(glfw.UI)
	canvas := u.CreateCanvas(screenWidth, screenHeight, screenScale, title)

	textureFactory := u.TextureFactory()
	game := blocks.NewGame(NewTextures(textureFactory))
	tick := time.Tick(frameTime)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	u.Start()
	defer u.Terminate()
	for {
		u.DoEvents()
		select {
		default:
			canvas.Draw(game.Draw)
		case <-tick:
			game.Update(canvas.InputState())
			if canvas.IsClosed() {
				return
			}
		case <-sigterm:
			return
		}
	}
}

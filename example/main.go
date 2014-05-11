package main

import (
	"github.com/hajimehoshi/go-ebiten/example/blocks"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Game interface {
	HandleEvent(e interface{})
	Update()
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

	u := cocoa.UI()
	window := u.CreateGameWindow(screenWidth, screenHeight, screenScale, title)

	windowEvents := window.Events()
	textureFactory := cocoa.TextureFactory()
	var game Game = blocks.NewGame(NewTextures(textureFactory))
	tick := time.Tick(frameTime)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	u.Start()
	defer u.Terminate()
	for {
		u.DoEvents()
		select {
		default:
			window.Draw(func(context graphics.Context) {
				game.Draw(context)
			})
		case <-tick:
			game.Update()
		case e := <-windowEvents:
			game.HandleEvent(e)
			if _, ok := e.(ui.WindowClosedEvent); ok {
				return
			}
		case <-sigterm:
			return
		}
	}
}

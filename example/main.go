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
	const title = "Ebiten Demo"

	u := cocoa.UI()
	textureFactory := cocoa.TextureFactory()
	window := u.CreateGameWindow(screenWidth, screenHeight, screenScale, title)

	drawing := make(chan struct{})
	quit := make(chan struct{})
	go func() {
		defer close(quit)

		textureFactoryEvents := textureFactory.Events()
		windowEvents := window.Events()
		var game Game = blocks.NewGame(textureFactory)
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			select {
			case e := <-textureFactoryEvents:
				game.HandleEvent(e)
			case e := <-windowEvents:
				game.HandleEvent(e)
				if _, ok := e.(ui.WindowClosedEvent); ok {
					return
				}
			case <-tick:
				game.Update()
			case <-drawing:
				window.Draw(func(context graphics.Context) {
					game.Draw(context)
				})
				drawing <- struct{}{}
			}
		}
	}()

	u.Start()
	defer u.Terminate()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	for {
		u.DoEvents()
		select {
		default:
			drawing <- struct{}{}
			<-drawing
		case <-s:
			return
		case <-quit:
			return
		}
	}
}

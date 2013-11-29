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
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"os"
	"runtime"
	"sync"
	"time"
)

type Game interface {
	InitTextures(tf graphics.TextureFactory)
	Update(context ebiten.GameContext)
	Draw(canvas graphics.Canvas)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	gameName := ""
	if 2 <= len(os.Args) {
		gameName = os.Args[1]
	}

	var game Game
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
	const fps = 60
	const title = "Ebiten Demo"

	var ui ebiten.UI = cocoa.New(screenWidth, screenHeight, screenScale, title)
	ui.InitTextures(game.InitTextures)

	lock := sync.Mutex{}
	go func() {
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			<-tick
			ui.Update(func(c ebiten.GameContext) {
				lock.Lock()
				defer lock.Unlock()
				game.Update(c)
			})
		}
	}()

	inputStateUpdated := ui.InputStateUpdated()
	for {
		ui.PollEvents()
	events:
		for {
			select {
			case e := <-inputStateUpdated:
				type InputStateUpdatedHandler interface {
					InputStateUpdated() chan<- ebiten.InputStateUpdatedEvent
				}
				if game2, ok := game.(InputStateUpdatedHandler); ok {
					game2.InputStateUpdated() <- e
				}
			default:
				break events
			}
		}
		ui.Draw(func(c graphics.Canvas) {
			lock.Lock()
			defer lock.Unlock()
			game.Draw(c)
		})
	}
}

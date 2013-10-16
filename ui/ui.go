package ui

import (
	"github.com/hajimehoshi/go-ebiten"
	"time"
)

type UI interface {
	MainLoop()
	ScreenWidth() int
	ScreenHeight() int
	Initializing() chan<- ebiten.Game
	Initialized() <-chan ebiten.Game
	Updating() chan<- ebiten.Game
	Updated() <-chan ebiten.Game
	Input() <-chan ebiten.InputState
}

func mainLoop(ui UI, game ebiten.Game) {
	ui.Initializing() <- game
	game = <-ui.Initialized()

	frameTime := time.Duration(int64(time.Second) / int64(ebiten.FPS))
	tick := time.Tick(frameTime)
	gameContext := &GameContext{
		screenWidth:  ui.ScreenWidth(),
		screenHeight: ui.ScreenHeight(),
		inputState:   ebiten.InputState{-1, -1},
	}
	for {
		select {
		case gameContext.inputState = <-ui.Input():
		case <-tick:
			game.Update(gameContext)
		case ui.Updating() <- game:
			game = <-ui.Updated()
		}
	}
}

func Run(ui UI, game ebiten.Game) {
	go mainLoop(ui, game)
	ui.MainLoop()
}

type GameContext struct {
	screenWidth  int
	screenHeight int
	inputState   ebiten.InputState
}

func (context *GameContext) ScreenWidth() int {
	return context.screenWidth
}

func (context *GameContext) ScreenHeight() int {
	return context.screenHeight
}

func (context *GameContext) InputState() ebiten.InputState {
	return context.inputState
}

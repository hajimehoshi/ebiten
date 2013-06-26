package ebiten

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"runtime"
	"time"
)

type TapInfo struct {
	X int
	Y int
}

type Game interface {
	ScreenWidth() int
	ScreenHeight() int
	Fps() int
	Init(tf graphics.TextureFactory)
	Update(input InputState)
	Draw(g graphics.GraphicsContext, offscreen graphics.Texture)
}

type UI interface {
	Run(device graphics.Device)
}

type InputState struct {
	IsTapped bool
	X        int
	Y        int
}

func mainLoop(game Game, input <-chan InputState, draw <-chan chan graphics.Drawable) {
	frameTime := time.Duration(int64(time.Second) / int64(game.Fps()))
	update := time.Tick(frameTime)
	for {
		select {
		case <-update:
			inputState := <-input
			game.Update(inputState)
		case gameDraw := <-draw:
			gameDraw <- game
			// TODO: wait!
		}
	}
}

func Run(game Game, ui UI,
	screenScale int,
	graphicsDevice graphics.Device,
	input <-chan InputState) {

	draw := graphicsDevice.Drawing()

	go mainLoop(game, input, draw)

	// UI should be executed on the main thread.
	ui.Run(graphicsDevice)
}

func init() {
	runtime.LockOSThread()
}

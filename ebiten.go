package ebiten

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/opengl"
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

func mainLoop(game Game, input <-chan InputState,
	deviceUpdate chan bool,
	gameDraw chan func(graphics.GraphicsContext, graphics.Texture)) {
	frameTime := time.Duration(int64(time.Second) / int64(game.Fps()))
	updateTick := time.Tick(frameTime)
	for {
		select {
		case <-updateTick:
			inputState := <-input
			game.Update(inputState)
		case <-deviceUpdate:
			ch := make(chan interface{})
			gameDraw <- func(g graphics.GraphicsContext,
				offscreen graphics.Texture) {
				game.Draw(g, offscreen)
				close(ch)
			}
			<-ch
		}
	}
}

func OpenGLRun(game Game, ui UI, screenScale int, input <-chan InputState) {
	deviceUpdate := make(chan bool)
	gameDraw := make(chan func(graphics.GraphicsContext, graphics.Texture))

	graphicsDevice := opengl.NewDevice(
		game.ScreenWidth(), game.ScreenHeight(),
		screenScale, deviceUpdate, gameDraw)

	game.Init(graphicsDevice.TextureFactory())

	go mainLoop(game, input, deviceUpdate, gameDraw)

	// UI should be executed on the main thread.
	ui.Run(graphicsDevice)
}

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

func OpenGLRun(game Game, ui UI, screenScale int, input <-chan InputState) {
	deviceUpdate := make(chan bool)
	commandSet := make(chan chan func(graphics.GraphicsContext))

	graphicsDevice := opengl.NewDevice(
		game.ScreenWidth(), game.ScreenHeight(), screenScale, deviceUpdate, commandSet)

	game.Init(graphicsDevice.TextureFactory())

	go func() {
		frameTime := time.Duration(int64(time.Second) / int64(game.Fps()))
		updateTick := time.Tick(frameTime)
		for {
			select {
			case <-updateTick:
				inputState := <-input
				game.Update(inputState)
			case <-deviceUpdate:
				commands := make(chan func(graphics.GraphicsContext))
				commandSet <- commands
				g := graphics.NewAsyncGraphicsContext(commands)
				// TODO: graphicsDevice is shared by multiple goroutines.
				game.Draw(g, graphicsDevice.OffscreenTexture())
				close(commands)
			}
		}
	}()


	// UI should be executed on the main thread.
	ui.Run(graphicsDevice)
}

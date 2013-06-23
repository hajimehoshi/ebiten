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
	Update()
	Draw(g graphics.GraphicsContext, offscreen graphics.Texture)
}

type UI interface {
	Run(device graphics.Device)
}

func OpenGLRun(game Game, ui UI, screenScale int) {
	ch := make(chan bool, 1)
	graphicsDevice := opengl.NewDevice(
		game.ScreenWidth(), game.ScreenHeight(), screenScale,
		func(g graphics.GraphicsContext, offscreen graphics.Texture) {
			ticket := <-ch
			game.Draw(g, offscreen)
			ch <- ticket
		})

	go func() {
		frameTime := time.Duration(int64(time.Second) / int64(game.Fps()))
		tick := time.Tick(frameTime)
		for {
			<-tick
			ticket := <-ch
			game.Update()
			ch <- ticket
		}
	}()

	game.Init(graphicsDevice.TextureFactory())
	ch <- true
	ui.Run(graphicsDevice)
}

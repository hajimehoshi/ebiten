package ebiten

import (
	"time"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
)

type Game interface {
	Init(tf graphics.TextureFactory)
	Update()
	Draw(g graphics.GraphicsContext, offscreen graphics.TextureID)
}

type UI interface {
	ScreenWidth() int
	ScreenHeight() int
	ScreenScale() int
	Run(device graphics.Device)
}

func OpenGLRun(game Game, ui UI) {
	ch := make(chan bool, 1)
	device := opengl.NewDevice(
		ui.ScreenWidth(), ui.ScreenHeight(), ui.ScreenScale(),
		func(g graphics.GraphicsContext, offscreen graphics.TextureID) {
			ticket := <-ch
			game.Draw(g, offscreen)
			ch<- ticket
		})

	go func() {
		const frameTime = time.Second / 60
		tick := time.Tick(frameTime)
		for {
			<-tick
			ticket := <-ch
			game.Update()
			ch<- ticket
		}
	}()

	game.Init(device.TextureFactory())
	ch<- true
	ui.Run(device)
}

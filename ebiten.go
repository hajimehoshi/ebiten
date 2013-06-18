package ebiten

import (
	"time"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
)

type Game interface {
	Update()
	Draw(g *graphics.GraphicsContext, offscreen *graphics.Texture)
}

func Run(game Game, u ui.UI) {
	ch := make(chan bool, 1)
	device := graphics.NewDevice(
		u.ScreenWidth(), u.ScreenHeight(), u.ScreenScale(),
		func(g *graphics.GraphicsContext, offscreen *graphics.Texture) {
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
	ch<- true

	u.Run(device)
}

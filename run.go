package ebiten

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Game interface {
	Update() error
	Draw(context GraphicsContext) error
}

func Run(u UI, game Game, width, height, scale int, title string, fps int) error {
	canvas, err := u.Start(width, height, scale, title)
	if err != nil {
		return err
	}

	frameTime := time.Duration(int64(time.Second) / int64(fps))
	tick := time.Tick(frameTime)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	defer u.Terminate()
	for {
		u.DoEvents()
		select {
		default:
			if err := canvas.Draw(game); err != nil {
				return err
			}
		case <-tick:
			if err := game.Update(); err != nil {
				return err
			}
			if canvas.IsClosed() {
				return nil
			}
		case <-sigterm:
			return nil
		}
	}
}

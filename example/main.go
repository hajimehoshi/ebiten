package main

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"image"
	_ "image/png"
	"os"
	"runtime"
	"time"
)

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()

	const screenWidth = 256
	const screenHeight = 240
	const screenScale = 2
	const fps = 60
	const title = "Ebiten Demo"

	type UI interface {
		ui.UI
		graphics.TextureFactory
	}
	var u UI = cocoa.New(screenWidth, screenHeight, screenScale, title)

	textureCreated := u.TextureCreated()
	renderTargetCreated := u.RenderTargetCreated()
	inputStateUpdated := u.InputStateUpdated()
	screenSizeUpdated := u.ScreenSizeUpdated()

	for tag, path := range TexturePaths {
		tag := tag
		path := path
		go func() {
			img, err := loadImage(path)
			if err != nil {
				panic(err)
			}
			u.CreateTexture(tag, img)
		}()
	}

	for tag, size := range RenderTargetSizes {
		tag := tag
		size := size
		go func() {
			u.CreateRenderTarget(tag, size.Width, size.Height)
		}()
	}

	drawing := make(chan *graphics.LazyCanvas)
	go func() {
		game := NewGame()
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			select {
			case e := <-textureCreated:
				game.OnTextureCreated(e)
			case e := <-renderTargetCreated:
				game.OnRenderTargetCreated(e)
			case e := <-inputStateUpdated:
				game.OnInputStateUpdated(e)
			case _, ok := <-screenSizeUpdated:
				if ok {
					// Do nothing
				} else {
					screenSizeUpdated = nil
				}
			case <-tick:
				game.Update()
			case canvas := <-drawing:
				game.Draw(canvas)
				drawing <- canvas
			}
		}
	}()

	for {
		u.PollEvents()
		u.Draw(func(actualCanvas graphics.Canvas) {
			drawing <- graphics.NewLazyCanvas()
			canvas := <-drawing
			canvas.Flush(actualCanvas)
		})
	}
}

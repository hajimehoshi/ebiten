package main

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"image"
	_ "image/png"
	"os"
	"runtime"
	"time"
)

func init() {
	runtime.LockOSThread()
}

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
	const screenWidth = 256
	const screenHeight = 240
	const screenScale = 3
	const fps = 60
	const title = "Ebiten Demo"

	ui := cocoa.UI()
	textureFactory := cocoa.TextureFactory()
	window := ui.CreateWindow(screenWidth, screenHeight, screenScale, title)

	textureCreated := textureFactory.TextureCreated()
	renderTargetCreated := textureFactory.RenderTargetCreated()

	for tag, path := range TexturePaths {
		tag := tag
		path := path
		go func() {
			img, err := loadImage(path)
			if err != nil {
				panic(err)
			}
			textureFactory.CreateTexture(tag, img)
		}()
	}

	for tag, size := range RenderTargetSizes {
		tag := tag
		size := size
		go func() {
			textureFactory.CreateRenderTarget(tag, size.Width, size.Height)
		}()
	}

	drawing := make(chan *graphics.LazyContext)
	quit := make(chan struct{})
	go func() {
		defer close(quit)

		mouseStateUpdated := window.MouseStateUpdated()
		screenSizeUpdated := window.ScreenSizeUpdated()
		windowClosed := window.WindowClosed()
		game := NewGame()
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			select {
			case e := <-textureCreated:
				game.OnTextureCreated(e)
			case e := <-renderTargetCreated:
				game.OnRenderTargetCreated(e)
			case e := <-mouseStateUpdated:
				game.OnMouseStateUpdated(e)
			case _, ok := <-screenSizeUpdated:
				if !ok {
					screenSizeUpdated = nil
				}
			case <-windowClosed:
				return
			case <-tick:
				game.Update()
			case context := <-drawing:
				game.Draw(context)
				drawing <- context
			}
		}
	}()

	for {
		ui.PollEvents()
		select {
		default:
			drawing <- graphics.NewLazyContext()
			context := <-drawing

			window.Draw(func(actualContext graphics.Context) {
				context.Flush(actualContext)
			})
			after := time.After(time.Duration(int64(time.Second) / 120))
			ui.PollEvents()
			<-after
		case <-quit:
			return
		}
	}
}

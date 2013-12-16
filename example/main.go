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

	u := cocoa.UI()
	textureFactory := cocoa.TextureFactory()
	window := u.CreateWindow(screenWidth, screenHeight, screenScale, title)

	textureFactoryEvents := textureFactory.Events()

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

		windowEvents := window.Events()
		game := NewGame()
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			select {
			case e := <-textureFactoryEvents:
				game.HandleEvent(e)
			case e := <-windowEvents:
				if _, ok := e.(ui.WindowClosedEvent); ok {
					return
				}
				game.HandleEvent(e)
			case <-tick:
				game.Update()
			case context := <-drawing:
				game.Draw(context)
				drawing <- context
			}
		}
	}()

	for {
		u.PollEvents()
		select {
		default:
			drawing <- graphics.NewLazyContext()
			context := <-drawing

			window.Draw(func(actualContext graphics.Context) {
				context.Flush(actualContext)
			})
			after := time.After(time.Duration(int64(time.Second) / 120))
			u.PollEvents()
			<-after
		case <-quit:
			return
		}
	}
}

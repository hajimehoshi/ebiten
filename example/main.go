package main

import (
	"github.com/hajimehoshi/go-ebiten/example/game/blank"
	"github.com/hajimehoshi/go-ebiten/example/game/input"
	"github.com/hajimehoshi/go-ebiten/example/game/monochrome"
	"github.com/hajimehoshi/go-ebiten/example/game/rects"
	"github.com/hajimehoshi/go-ebiten/example/game/rotating"
	"github.com/hajimehoshi/go-ebiten/example/game/sprites"
	"github.com/hajimehoshi/go-ebiten/example/game/testpattern"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
	"github.com/hajimehoshi/go-ebiten/ui/cocoa"
	"image"
	"os"
	"runtime"
	"time"
)

type Game interface {
	InitTextures(tf graphics.TextureFactory)
	Update()
	Draw(canvas graphics.Canvas)
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
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()

	gameName := ""
	if 2 <= len(os.Args) {
		gameName = os.Args[1]
	}

	var game Game
	switch gameName {
	case "blank":
		game = blank.New()
	case "input":
		game = input.New()
	case "monochrome":
		game = monochrome.New()
	case "rects":
		game = rects.New()
	default:
		fallthrough
	case "rotating":
		game = rotating.New()
	case "sprites":
		game = sprites.New()
	case "testpattern":
		game = testpattern.New()
	}

	const screenWidth = 256
	const screenHeight = 240
	const screenScale = 2
	const fps = 60
	const title = "Ebiten Demo"

	type UI interface {
		ui.UI
		graphics.TextureFactory2
	}
	var u UI = cocoa.New(screenWidth, screenHeight, screenScale, title)

	// TODO: Remove this
	u.LoadResources(game.InitTextures)

	textureCreated := u.TextureCreated()
	inputStateUpdated := u.InputStateUpdated()
	screenSizeUpdated := u.ScreenSizeUpdated()

	img, err := loadImage("images/ebiten.png")
	if err != nil {
		panic(err)
	}
	u.CreateTexture("ebiten", img)

	drawing := make(chan *graphics.LazyCanvas)
	go func() {
		frameTime := time.Duration(int64(time.Second) / int64(fps))
		tick := time.Tick(frameTime)
		for {
			select {
			case e, ok := <-textureCreated:
				if ok {
					print(e.Error)
				} else {
					textureCreated = nil
				}
			case e, ok := <-inputStateUpdated:
				// TODO: Use Adaptor?
				if ok {
					if game2, ok := game.(interface {
						OnInputStateUpdated(ui.InputStateUpdatedEvent)
					}); ok {
						game2.OnInputStateUpdated(e)
					}
				} else {
					inputStateUpdated = nil
				}
			case e, ok := <-screenSizeUpdated:
				if ok {
					if game2, ok := game.(interface {
						OnScreenSizeUpdated(ui.ScreenSizeUpdatedEvent)
					}); ok {
						game2.OnScreenSizeUpdated(e)
					}
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

package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten"
	"image/color"
	"log"
)

var (
	xWidth          int
	yHeight         int
	alivePercentage int
	world           *World
)

func main() {
	flag.IntVar(&xWidth, "width", 480, "-width=640")
	flag.IntVar(&yHeight, "height", 320, "-height=480")
	flag.IntVar(&alivePercentage, "percent", 20, "-percent=30")
	flag.Parse()

	//create world
	world = NewWorld(xWidth, yHeight)
	world.GenerateGrid(alivePercentage)

	if err := ebiten.Run(update, xWidth, yHeight, 2, "Hello, World!"); err != nil {
		log.Fatal(err)
	}
}

func update(screen *ebiten.Image) error {
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	err := screen.Fill(color.RGBA{0, 0, 0, 0xff})
	if err != nil {
		return err
	}

	background, err := ebiten.NewImage(xWidth, yHeight, ebiten.FilterDefault)
	if err != nil {
		return err
	}

	world.Print(background)
	world.Next()

	err = screen.DrawImage(background, &ebiten.DrawImageOptions{})
	if err != nil {
		return err
	}
	return nil
}

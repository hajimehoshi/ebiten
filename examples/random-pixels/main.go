package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	offscreen *ebiten.Image
}

func NewGame() *Game {
	return &Game{
		offscreen: ebiten.NewImage(screenWidth, screenHeight),
	}
}

func (g *Game) Update() error {
	s := g.offscreen.Bounds().Size()
	x := rand.IntN(s.X)
	y := rand.IntN(s.Y)
	c := color.RGBA{
		byte(rand.IntN(256)),
		byte(rand.IntN(256)),
		byte(rand.IntN(256)),
		byte(0xff),
	}
	g.offscreen.Set(x, y, c)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.offscreen, nil)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Set (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

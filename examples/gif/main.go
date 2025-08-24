package main

import (
    "log"
    "image/color"

    "github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
    animator *GIFAnimator
}

func (g *Game) Update() error {
    g.animator.Update()
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    screen.Fill(color.Black)
    g.animator.Draw(screen, 0, 0)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    return 640, 480
}

func main() {
    animator, err := NewGIFAnimator("dragon.gif")
    if err != nil {
        log.Fatal(err)
    }

    game := &Game{animator: animator}
    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("GIF in Ebiten")
    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}

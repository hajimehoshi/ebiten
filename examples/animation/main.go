package main

import (
    "log"
    "github.com/hajimehoshi/ebiten/v2"
)

var currentScene Scene

type Scene interface {
    Update() error
    Draw(screen *ebiten.Image)
}

func main() {
    currentScene = &Scene1{
                    count: 0,
                    t:     0,
                    tDir:  1,
                }

    ebiten.SetWindowSize(screenWidth * 2, screenHeight * 2)
    ebiten.SetWindowTitle("Local Karate Minus")
    if err := ebiten.RunGame(&Game{}); err != nil {
        log.Fatal(err)
    }
}

type Game struct{}

func (g *Game) Update() error {
    return currentScene.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
    currentScene.Draw(screen)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
    return screenWidth * 2, screenHeight * 2
}

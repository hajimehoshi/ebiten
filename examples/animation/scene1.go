package main

import (
    _ "image/png"
    "image/color"
    "log"

    "github.com/hajimehoshi/ebiten/v2"
)

type Scene1 struct {
    count int
    t     float64
    tDir  float64
}

func (s *Scene1) Update() error {
    s.count++

    moveSprite()
    moveBall(s, &circleX, &circleY)

     if posX > screenWidth {
         currentScene = &Scene2{}
         log.Println("Scene2 loaded")
         posX = float64(3)
     }

    return nil
}

func (s *Scene1) Draw(screen *ebiten.Image) {
    screen.Fill(color.RGBA{211, 211, 211, 255})
    drawBackground(screen, backgroundImage1)
    drawSprite(s.count, screen)
    drawBackground(screen, backgroundImage2)

    circleImage := createCircleImage(8, color.White)
    op2 := &ebiten.DrawImageOptions{}
    op2.GeoM.Translate(circleX-8, circleY-8)
    op2.GeoM.Scale(2, 2)
    screen.DrawImage(circleImage, op2)
}

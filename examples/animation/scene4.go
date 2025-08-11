package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "image/color"
    "log"
    )

type Scene4 struct {
    count int
    t     float64
    tDir  float64
}

func (s *Scene4) Update() error {
    s.count++
    moveSprite()
    moveBall(s, &circleX, &circleY)

     if posX < 0 {
         currentScene = &Scene3{}
         log.Println("Scene3 loaded")
         posX = float64(screenWidth - 3)
     } else if posX > screenWidth {
         spiderX = float64(2*screenWidth)
         currentScene = &Scene5{
            }
         log.Println("Scene5 loaded")
         posX = float64(3)
     }
    return nil
}

func (s *Scene4) Draw(screen *ebiten.Image) {
    drawBackground(screen, yieArKF)
    drawSprite(s.count, screen)

    circleImage := createCircleImage(8, color.White)
    op2 := &ebiten.DrawImageOptions{}
    op2.GeoM.Translate(circleX-8, circleY-8)
    op2.GeoM.Scale(2, 2)
    screen.DrawImage(circleImage, op2)
}

package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "log"
    )

type Scene3 struct {
    count int
}

func (s *Scene3) Update() error {
    s.count++
    moveSprite()

     if posX < 0 {
         currentScene = &Scene2{}
         log.Println("Scene2 loaded")
         posX = float64(screenWidth - 3)
     } else if posX > screenWidth {
         currentScene = &Scene4{
                count: 0,
                t:     0,
                tDir:  1,
            }
         log.Println("Scene4 loaded")
         posX = float64(3)
     }
    return nil
}

func (s *Scene3) Draw(screen *ebiten.Image) {
    drawBackground(screen, newYork)
    drawSprite(s.count, screen)
}

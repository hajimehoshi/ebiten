package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "log"
    )

type Scene2 struct {
    count int
}

func (s *Scene2) Update() error {
    s.count++
    moveSprite()

     if posX < 0 {
         currentScene = &Scene1{
                count: 0,
                t:     0,
                tDir:  1,
            }
         log.Println("Scene1 loaded")
         posX = float64(screenWidth - 3)
     } else if posX > screenWidth {
         currentScene = &Scene3{
            }
         log.Println("Scene3 loaded")
         posX = float64(3)
     }
    return nil
}

func (s *Scene2) Draw(screen *ebiten.Image) {
    drawBackground(screen, backgroundImage3)
    drawSprite(s.count, screen)
}

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

func (s *Scene4) GetT() float64 { return s.t }
func (s *Scene4) SetT(val float64) { s.t = val }

func (s *Scene4) GetTDir() float64 { return s.tDir }
func (s *Scene4) SetTDir(val float64) { s.tDir = val }

func (s *Scene4) GetCount() int { return s.count }

func (s *Scene4) Update() error {
    s.count++
    moveSprite()
    moveBall(s, &circleX, &circleY)

     if posX < 0 {
         currentScene = &Scene3{}
         log.Println("Scene3 loaded")
         posX = float64(screenWidth - 3)
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

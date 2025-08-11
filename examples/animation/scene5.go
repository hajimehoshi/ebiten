package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "log"
    "image"
    )

type Scene5 struct {
    count int
}

func (s *Scene5) Update() error {
    s.count++
    moveSprite()
    moveSpider()

     if posX < 0 {
         currentScene = &Scene4{
                count: 0,
                t:     0,
                tDir:  1,
            }
         log.Println("Scene4 loaded")
         posX = float64(screenWidth - 3)
     }
    return nil
}

func moveSpider(){
    spiderX -= float64(1)
}

func drawSpider(count int, screen *ebiten.Image) {
    /*frameHeight := 180
    i := (count / 10) % 13
    sx := 0
    sy := i * frameHeight
    spriteRect := image.Rect(sx, sy, sx+250, sy+frameHeight)
    spriteSubImage := spiderImage.SubImage(spriteRect).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Reset()
    op.GeoM.Translate(spiderX, 7*screenHeight /8)
    screen.DrawImage(spriteSubImage, op)
    */
    frameHeight := 180
    frameWidth := 250
    i := (count / 5) % 13
    sx := i * int(frameWidth)
    sy := 0
    spriteRect := image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)
    spriteSubImage := spiderImage.SubImage(spriteRect).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Reset()
    op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2)
    op.GeoM.Translate(spiderX, 2.5*posY)
    //op.GeoM.Scale(2, 2)
    screen.DrawImage(spriteSubImage, op)
}

func (s *Scene5) Draw(screen *ebiten.Image) {
    drawBackground(screen, sidney)
    drawSprite(s.count, screen)
    drawSpider(s.count, screen)
}

package main

import (
    "image"
    _ "image/png"
    "image/color"
    "log"
    "os"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/audio"
    "github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type BallScene interface {
    GetT() float64
    SetT(float64)
    GetTDir() float64
    SetTDir(float64)
    GetCount() int
}

type Scene1 struct {
    count int
    t     float64
    tDir  float64
}

func (s *Scene1) GetT() float64 { return s.t }
func (s *Scene1) SetT(val float64) { s.t = val }

func (s *Scene1) GetTDir() float64 { return s.tDir }
func (s *Scene1) SetTDir(val float64) { s.tDir = val }

func (s *Scene1) GetCount() int { return s.count }

const (
    screenWidth  = 474
    screenHeight = 299

    frameOX     = 0
    frameOY     = 0
    frameWidth  = 55
    frameHeight = 71
    frameCount  = 3
)

var (
    runnerImage      *ebiten.Image
    backgroundImage1 *ebiten.Image
    backgroundImage2 *ebiten.Image
    backgroundImage3 *ebiten.Image
    newYork          *ebiten.Image
    yieArKF          *ebiten.Image

    posX = float64(474) * 0.4
    posY = float64(220 - 71/2)

    circleX = float64(474) * 0.5
    circleY = float64(299) * 0.8

    movement = float64(3)

    context *audio.Context
    player  *audio.Player
)

func init() {
    // Load assets here (replace with your actual image paths)
    var err error
    runnerImage, err = loadImage("kw1.png")
    if err != nil {
        log.Fatal(err)
    }
    backgroundImage1, err = loadImage("background.png")
    if err != nil {
        log.Fatal(err)
    }
    backgroundImage2, err = loadImage("background2.png")
    if err != nil {
        log.Fatal(err)
    }
    backgroundImage3, err = loadImage("background3.png")
    if err != nil {
        log.Fatal(err)
    }
    newYork, err = loadImage("ny.png")
    if err != nil {
        log.Fatal(err)
    }
    yieArKF, err = loadImage("yie-ar.png")
    if err != nil {
        log.Fatal(err)
    }
    if err := initAudio(); err != nil {
        log.Fatal(err)
    }
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

func moveBall(s BallScene, circleX *float64, circleY *float64) {
    *circleX += movement
    if *circleX > float64(screenWidth) || *circleX < 0 {
        movement *= -1
    }

    s.SetT(s.GetT() + s.GetTDir()*0.04)
    t := s.GetT()

    if t > 1 {
        t = 1
        s.SetT(t)
        s.SetTDir(-s.GetTDir())
        player.Rewind()
        player.Play()
    } else if t < 0 {
        t = 0
        s.SetT(t)
        s.SetTDir(-s.GetTDir())
        player.Rewind()
        player.Play()
    }

    a := (float64(screenHeight)*0.92 - float64(screenHeight)*0.65) / 0.25
    *circleY = a * (t - 0.5) * (t - 0.5) + float64(screenHeight)*0.7
}

func moveSprite() {
    if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
        posX -= 4
    } else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
        posX += 4
    }
}

func loadImage(path string) (*ebiten.Image, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    img, _, err := image.Decode(file)
    if err != nil {
        return nil, err
    }
    return ebiten.NewImageFromImage(img), nil
}

func initAudio() error {
    context = audio.NewContext(44100)
    f, err := os.Open("bounce.wav")
    if err != nil {
        return err
    }
    stream, err := wav.Decode(context, f)
    if err != nil {
        return err
    }
    player, err = audio.NewPlayer(context, stream)
    if err != nil {
        return err
    }
    return nil
}

func drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
    subImg := bg.SubImage(image.Rect(0, 0, 474, 299)).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Scale(2, 2)
    screen.DrawImage(subImg, op)
}

func drawSprite(count int, screen *ebiten.Image) {
    i := (count / 5) % 3
    sx := i * 55
    sy := 0
    spriteRect := image.Rect(sx, sy, sx+55, sy+71)
    spriteSubImage := runnerImage.SubImage(spriteRect).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Reset()
    op.GeoM.Translate(-55/2, -71/2)
    op.GeoM.Translate(posX, posY)
    op.GeoM.Scale(2, 2)
    screen.DrawImage(spriteSubImage, op)
}

func createCircleImage(radius int, col color.Color) *ebiten.Image {
    size := radius * 2
    img := ebiten.NewImage(size, size)
    img.Fill(color.Transparent)
    for y := -radius; y <= radius; y++ {
        for x := -radius; x <= radius; x++ {
            if x*x+y*y <= radius*radius {
                img.Set(x+radius, y+radius, col)
            }
        }
    }
    return img
}

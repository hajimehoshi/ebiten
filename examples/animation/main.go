package main

import (
	"image"
	_ "image/png"
	_ "image/jpeg"
	"image/color"
	//"math"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	runnerImage *ebiten.Image
	backgroundImage *ebiten.Image

    posX = float64(screenWidth) * 0.4
    posY = float64(screenHeight) * 0.85
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
    		posX -= 2
    	} else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
    		posX += 2
    	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    // Fill background
    screen.Fill(color.RGBA{R: 211, G: 211, B: 211, A: 255})

    // Draw background image
    bgSubImage := backgroundImage.SubImage(image.Rect(0, 0, screenWidth, screenHeight)).(*ebiten.Image)
    screen.DrawImage(bgSubImage, &ebiten.DrawImageOptions{})

    // Calculate current frame index
    i := (g.count / 5) % frameCount

    // Calculate sprite sheet position
    sx := frameOX + i*frameWidth
    sy := frameOY


    // Prepare drawing options
    op := &ebiten.DrawImageOptions{}

    // Center the sprite at (posX, posY)
    op.GeoM.Reset()
    op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2)
    op.GeoM.Translate(posX, posY)

    // Define sprite crop rectangle
    spriteRect := image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)
    spriteSubImage := runnerImage.SubImage(spriteRect).(*ebiten.Image)

    // Draw sprite at the position
    screen.DrawImage(spriteSubImage, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
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

func main() {
    var err error

    ebiten.SetMaxTPS(20)
	runnerImage, err = loadImage("kw1.png")
    if err != nil {
        log.Fatal(err)
    }

    backgroundImage, err = loadImage("background.jpg")
    if err != nil {
        log.Fatal(err)
    }

	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Animation (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

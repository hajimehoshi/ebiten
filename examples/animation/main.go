package main

import (
	"image"
	_ "image/png"
	_ "image/jpeg"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/audio"
    "github.com/hajimehoshi/ebiten/v2/audio/wav"
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
	backgroundImage1 *ebiten.Image
	backgroundImage2 *ebiten.Image

    posX = float64(screenWidth) * 0.4
    posY = float64(220 - frameHeight/2)

    circleX = float64(screenWidth) / 2
    circleY = float64(screenHeight) * 0.8

    movement = float64(3)

    context     *audio.Context
    player      *audio.Player
)

type Game struct {
	count int
	t     float64
	tDir  float64
}

func (g *Game) Update() error {
	g.count++

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
    		posX -= 4
    	} else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
    		posX += 4
    	}

    circleX += movement

    if (circleX > screenWidth || circleX < 0){
        movement *= -1
        }
    g.t += g.tDir * 0.04
    if g.t > 1 {
        g.t = 1
        g.tDir = -g.tDir
        player.Rewind()
        player.Play()
    } else if g.t < 0 {
        g.t = 0
        g.tDir = -g.tDir
        player.Rewind()
        player.Play()
    }
    a := (float64(screenHeight)*0.92 - float64(screenHeight)*0.65) / 0.25 // height difference over parabola width
    circleY = a* (g.t - 0.5)*(g.t - 0.5) + float64(screenHeight)*0.7

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    screen.Fill(color.RGBA{R: 211, G: 211, B: 211, A: 255})

    drawBackground(screen, backgroundImage1)
    drawSprite(g, screen)
    drawBackground(screen, backgroundImage2)

    circleImage := createCircleImage(8, color.White)
    op2 := &ebiten.DrawImageOptions{}
    op2.GeoM.Translate(circleX - 8, circleY - 8)
    screen.DrawImage(circleImage, op2)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}


func drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
    subImg := bg.SubImage(image.Rect(0, 0, screenWidth, screenHeight)).(*ebiten.Image)
    screen.DrawImage(subImg, &ebiten.DrawImageOptions{})
}

func drawSprite(g *Game, screen *ebiten.Image) {
    i := (g.count / 5) % frameCount
    sx := frameOX + i*frameWidth
    sy := frameOY
    spriteRect := image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)
    spriteSubImage := runnerImage.SubImage(spriteRect).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Reset()
    op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2)
    op.GeoM.Translate(posX, posY)
    screen.DrawImage(spriteSubImage, op)
}

func createCircleImage(radius int, col color.Color) *ebiten.Image {
    size := radius * 2
    img := ebiten.NewImage(size, size)

    img.Fill(color.Transparent)

    for y := -radius; y <= radius; y++ {
        for x := -radius; x <= radius; x++ {
            if x*x + y*y <= radius*radius {
                img.Set(x+radius, y+radius, col)
            }
        }
    }
    return img
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
    //defer f.Close()
    return nil
}

func main() {
    var err error

    ebiten.SetMaxTPS(20)
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

	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Local Karate Minus")

    gameInstance := &Game{
        count: 0,
        t:     0,
        tDir:  1,
    }

    initAudio();

	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"os"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/audio"
    "github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
    err           error
	logo    *ebiten.Image
	background    *ebiten.Image
	mplusFaceSource *text.GoTextFaceSource
    context *audio.Context
    player  *audio.Player
    player2  *audio.Player
    counter float64

    shiftX int
    shiftY int
    currentStage int
)

const (
    stage1Timeout = 1900
    stage2Timeout = 26000

    final = stage2Timeout

    stage1MusicPath = "audio/Boards dont hit back.wav"
    stage2MusicPath = "audio/BruceLee.wav"

    moveSpeed = 2
)

func init() {
    shiftX = 0
    shiftY = 0

    context = audio.NewContext(44100)

    logo, err = loadImage("pics/bruce-lee3.png")
    if err != nil {
        log.Fatal(err)
    }

    background, err = loadImage("pics/brusli2.png")
    if err != nil {
        log.Fatal(err)
    }

	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

func initAudio(path string) (*audio.Player, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    stream, err := wav.Decode(context, f)
    if err != nil {
        return nil, err
    }

    localPlayer, err := audio.NewPlayer(context, stream)
    if err != nil {
        return nil, err
    }

    //defer f.Close()
    return localPlayer, nil
}

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++

	if (counter > final) {
        return ebiten.Termination
    }

	if runtime.GOOS == "js" {
		if ebiten.IsKeyPressed(ebiten.KeyF) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			ebiten.SetFullscreen(true)
		}
	}
    playbackDone := player == nil || !player.IsPlaying()

	if runtime.GOOS != "js" && (ebiten.IsKeyPressed(ebiten.KeyQ) || playbackDone) {
	    fmt.Print(counter)
		return nil
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    counter += 7
    if (counter < stage1Timeout) {
        stage1(screen)
    } else if (counter < stage2Timeout) {
        stage2(screen)
    }
}

func stage2(screen *ebiten.Image){
    drawBackground(screen, background, shiftX, shiftY, 2555, 705)

    if player2 == nil{
        player2, err = initAudio(stage2MusicPath)
        player2.Play()

        if err != nil {
        	log.Fatal(err)
        }
    }

    move()
}

func stage1(screen *ebiten.Image) {

	scale := ebiten.Monitor().DeviceScaleFactor()

	drawBackground(screen, logo, 20, 20, 410, 371)
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	fw, fh := ebiten.Monitor().Size()
	msg := ""
	if runtime.GOOS == "js" {
		msg += "Press F or touch the screen to enter fullscreen (again).\n"
	} else {
		msg += "Press Q to quit.\n"
	}
	msg += fmt.Sprintf("Screen size in fullscreen: %d, %d\n", fw, fh)
	msg += fmt.Sprintf("Game's screen size: %d, %d\n", sw, sh)
	msg += fmt.Sprintf("Device scale factor: %0.2f\n", scale)

	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(50*scale, 650*scale)
	textOp.ColorScale.ScaleWithColor(color.White)
	textOp.LineSpacing = 12 * ebiten.Monitor().DeviceScaleFactor() * 1.5
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   12 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)

	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   12 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)

    msg = story()


	textOp.GeoM.Translate(610*scale, (400-counter)*scale)
	textOp.LineSpacing = 30 * ebiten.Monitor().DeviceScaleFactor() * 2
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   30 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)
}

func move() {
    if (shiftX > moveSpeed) && (ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)) {
        shiftX -= moveSpeed
    } else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
        shiftX += moveSpeed
    }
    if (shiftY > moveSpeed) && (ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)) {
        shiftY -= moveSpeed
    } else if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
        shiftY += moveSpeed
    }
    fmt.Println(" [", shiftX, shiftY, "] ")
}

func story() string {
    return `BOARDS DON'T HIT BACK....`
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
    player, err = initAudio(stage1MusicPath)
    player.Play()

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Fullscreen (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

func drawBackground(screen, bg *ebiten.Image, x, y, w, h int) {
    subImg := bg.SubImage(image.Rect(x, y, w, h)).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Scale(2, 2)
    screen.DrawImage(subImg, op)
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

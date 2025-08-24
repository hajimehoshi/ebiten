package main

import (
	"bytes"
	"image"
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
    err         error
	logo        *ebiten.Image
	background  *ebiten.Image
	background2 *ebiten.Image
	pic1nun     *ebiten.Image
	pic2nun     *ebiten.Image
	pic3nun     *ebiten.Image
	pic4        *ebiten.Image
	projectileImg        *ebiten.Image
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
    stage1Timeout = 5000
    stage2Timeout = 1900 + stage1Timeout
    stage3Timeout = 24000 + stage2Timeout

    final = stage3Timeout

    stage2MusicPath = "audio/Boards dont hit back.wav"
    stage3MusicPath = "audio/BruceLee.wav"

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

    background2, err = loadImage("pics/brusli2.png")
    if err != nil {
        log.Fatal(err)
    }
    background, err = loadImage("pics/brusli.png")
    if err != nil {
        log.Fatal(err)
    }
    pic1nun, err = loadImage("pics/bruce-lee-nunchako1.png")
    if err != nil {
        log.Fatal(err)
    }
    pic2nun, err = loadImage("pics/bruce-lee-nunchako2.png")
    if err != nil {
        log.Fatal(err)
    }
    pic3nun, err = loadImage("pics/bruce-lee2.png")
    if err != nil {
        log.Fatal(err)
    }
    pic4, err = loadImage("pics/bl.png")
    if err != nil {
        log.Fatal(err)
    }
    projectileImg, err = loadImage("pics/projectile.png")
    if err != nil {
        log.Fatal(err)
    }

	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s

	initStage3()
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
	    //fmt.Print(counter)
		return nil
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    counter += 7
    if (counter < stage1Timeout) {
        stage1(screen, counter)
    }else if (counter < stage2Timeout) {
        stage2(screen)
    } else if (counter < stage3Timeout) {
        stage3(screen, counter)
    }
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
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

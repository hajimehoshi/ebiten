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
	background    *ebiten.Image
	mplusFaceSource *text.GoTextFaceSource
    context *audio.Context
    player  *audio.Player
    counter float64
)

func init() {
    background, err = loadImage("broken-land.png")
    if err != nil {
        log.Fatal(err)
    }
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

func initAudio() error {
    context = audio.NewContext(44100)
    f, err := os.Open("introduction.wav")
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

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++

	if runtime.GOOS == "js" {
		if ebiten.IsKeyPressed(ebiten.KeyF) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			ebiten.SetFullscreen(true)
		}
	}
    playbackDone := player == nil || !player.IsPlaying()
	if runtime.GOOS != "js" && (ebiten.IsKeyPressed(ebiten.KeyQ) || playbackDone ) {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    counter += 0.4
	scale := ebiten.Monitor().DeviceScaleFactor()

	drawBackground(screen, background)
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


	textOp.GeoM.Translate(500*scale, (400-counter)*scale)
	textOp.LineSpacing = 30 * ebiten.Monitor().DeviceScaleFactor() * 1.2
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   30 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)
}

func story() string {
    return `Darkness had fallen over the realm.

The land was cracked and scarred from endless war, as armies clashed
and kingdoms tumbled into ruin. Above, the skies teemed with
fire-breathing dragons, their wings blotting out the sun as they raided
villages, leaving ash and ruin in their wake.
Forests that once thrived with life now lay charred and silent,
their trees reduced to ash and memory.

In the shadows, goblins prowled through the wreckage, scavenging
what little remained of a fallen world. Hope flickered
faintly in the hearts of the survivors—if it was still there at all.

Was there any salvation for this broken land? Or had the darkness
claimed it forever?`
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

func main() {
    initAudio()
    player.Play()

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Fullscreen (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

func drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
    subImg := bg.SubImage(image.Rect(0, 0, 512, 512)).(*ebiten.Image)
    op := &ebiten.DrawImageOptions{}
    //op.GeoM.Scale(2, 2)
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

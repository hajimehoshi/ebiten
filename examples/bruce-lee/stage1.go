package main

import (
	"fmt"
	"log"
	"image/color"
	_ "image/jpeg"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func stage1(screen *ebiten.Image, counter float64) {
    var pic *ebiten.Image

    switch {
    case counter < 1000:
        pic = pic4
    case counter < 2000:
        pic = pic1nun
    case counter < 3000:
        pic = pic2nun
    case counter < 4000:
        pic = pic3nun
    default:
        pic = logo
    }

    pos :=  int((counter)/15)
    if (pic == logo || pic == pic3nun){
        pos = 20
    }

    drawBackground(screen, pic, pos - 200, pos-99, 2555, 705)

    if player == nil{
        player, err = initAudio(stage2MusicPath)
        player.Play()

        if err != nil {
        	log.Fatal(err)
        }
    }

	scale := ebiten.Monitor().DeviceScaleFactor()

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	fw, fh := ebiten.Monitor().Size()

	msg := fmt.Sprintf("Screen size in fullscreen: %d, %d\n", fw, fh)
	msg += fmt.Sprintf("Game's screen size: %d, %d\n", sw, sh)
	msg += fmt.Sprintf("Device scale factor: %0.2f\n", scale)
	msg += fmt.Sprintf("\nBeware of fat YAMO and the Ninja...\n")

	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(830*scale, 700*scale)
	textOp.ColorScale.ScaleWithColor(color.White)
	textOp.LineSpacing = 12 * ebiten.Monitor().DeviceScaleFactor() * 1.5
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   12 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)

    msg = story()
	textOp.GeoM.Translate(10*scale, (400-counter)*scale)
	textOp.LineSpacing = 30 * ebiten.Monitor().DeviceScaleFactor() * 2
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   30 * ebiten.Monitor().DeviceScaleFactor(),
	}, textOp)
}

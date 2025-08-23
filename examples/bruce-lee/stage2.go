package main

import (
	"image/color"
	_ "image/jpeg"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func stage2(screen *ebiten.Image) {

	scale := ebiten.Monitor().DeviceScaleFactor()

	drawBackground(screen, logo, 20 - 200, 20-99, 2555, 705)
    msg := "Entering castle of the SORCERER...\n"

	if runtime.GOOS == "js" {
		msg += "Press F or touch the screen to enter fullscreen (again).\n"
	}

	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(50*scale, 750*scale)
	textOp.ColorScale.ScaleWithColor(color.White)
	textOp.LineSpacing = 12 * ebiten.Monitor().DeviceScaleFactor() * 1.5
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


func story() string {
    return `BOARDS DON'T HIT BACK....`
}

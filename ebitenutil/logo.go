package ebitenutil

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawLogo draws the Ebiten logo on the image.
func DrawLogo(dst *ebiten.Image) {
	var (
		bounds     = dst.Bounds()
		maxSize    = math.Min(float64(bounds.Dx()), float64(bounds.Dy()))
		logoSize   = int(maxSize * 0.16)
		totalSize  = int(float64(logoSize) * 2.778)
		logoOffset = int(float64(logoSize) * (4.0 / 9.0))
		tailWidth  = int(float64(logoSize) * (5.0 / 9.0))
		x          = (bounds.Dx() / 2) - (totalSize / 2)
		y          = bounds.Dy() / 2
		logoColor  = color.RGBA{219, 86, 32, 255}
	)

	// Draw ebiten.
	for i := 0; i < 3; i++ {
		offset := i * logoOffset
		dst.SubImage(image.Rect(x+offset, y-offset, x+logoSize+offset, y+logoSize-offset)).(*ebiten.Image).Fill(logoColor)
	}
	offset := 4 * logoOffset
	dst.SubImage(image.Rect(x+offset, y-offset, x+tailWidth+offset, y+logoSize-offset)).(*ebiten.Image).Fill(logoColor)
	dst.SubImage(image.Rect(x+offset+logoOffset, y-offset+logoOffset, x+offset+logoSize, y-offset+logoSize)).(*ebiten.Image).Fill(logoColor)

	// Draw text.
	if maxSize < 225 {
		return
	}
	const logoText = "POWERED BY EBITEN"
	logoTextScale := float64(logoSize) * .0275
	logoTextWidth := 6.0 * float64(len(logoText)) * logoTextScale
	img := ebiten.NewImage(200, 200)
	DebugPrint(img, logoText)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(logoTextScale, logoTextScale)
	op.GeoM.Translate(float64(bounds.Dx())/2-float64(logoTextWidth)/2-(logoTextScale*4), float64(bounds.Dy())/2+float64(logoSize)+(logoTextScale*4))
	dst.DrawImage(img, op)
}

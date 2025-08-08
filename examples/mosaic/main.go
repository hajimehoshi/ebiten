package main

import (
	"os"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

const mosaicRatio = 16

var (
	gophersImage *ebiten.Image
)

func init() {
		file, err := os.Open("../resources/images/gophers.jpg")
            if err != nil {
                log.Fatal(err)
            }
            defer file.Close()

    	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)
}

type Game struct {
	gophersRenderTarget *ebiten.Image
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Shrink the image once.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/mosaicRatio, 1.0/mosaicRatio)
	g.gophersRenderTarget.DrawImage(gophersImage, op)

	// Enlarge the shrunk image.
	// The filter is the nearest filter, so the result will be mosaic.
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(mosaicRatio, mosaicRatio)
	screen.DrawImage(g.gophersRenderTarget, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	g := &Game{
		gophersRenderTarget: ebiten.NewImage(w/mosaicRatio, h/mosaicRatio),
	}
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Mosaic (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

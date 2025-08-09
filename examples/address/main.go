package main

import (
	"os"
	"image"
	_ "image/png"
	"log"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var ebitenImage *ebiten.Image

func init() {
    fmt.Println("Run within 'address' directory (since relative paths are used in the code.)")
	img, err := loadImage("../resources/images/ebiten.png")
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)
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

func drawRect(screen *ebiten.Image, img *ebiten.Image, x, y, width, height float32, address ebiten.Address, msg string) {
	sx, sy := -width/2, -height/2
	vs := []ebiten.Vertex{
		{
			DstX:   x,
			DstY:   y,
			SrcX:   sx,
			SrcY:   sy,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x + width,
			DstY:   y,
			SrcX:   sx + width,
			SrcY:   sy,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x,
			DstY:   y + height,
			SrcX:   sx,
			SrcY:   sy + height,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   x + width,
			DstY:   y + height,
			SrcX:   sx + width,
			SrcY:   sy + height,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = address
	screen.DrawTriangles(vs, []uint16{0, 1, 2, 1, 2, 3}, img, op)

	ebitenutil.DebugPrintAt(screen, msg, int(x), int(y)-16)
}

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	const ox, oy = 40, 60
	drawRect(screen, ebitenImage, ox, oy, 200, 100, ebiten.AddressClampToZero, "Regular")
	drawRect(screen, ebitenImage, 220+ox, oy, 200, 100, ebiten.AddressRepeat, "Regular, Repeat")

	subImage := ebitenImage.SubImage(image.Rect(10, 5, 20, 30)).(*ebiten.Image)
	drawRect(screen, subImage, ox, 200+oy, 200, 100, ebiten.AddressClampToZero, "Subimage")
	drawRect(screen, subImage, 220+ox, 200+oy, 200, 100, ebiten.AddressRepeat, "Subimage, Repeat")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Sampler Address (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

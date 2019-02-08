package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten"
)

const (
	screenWidth  = 100 // 320
	screenHeight = 50  // 240
)

type palete struct {
	r uint8
	g uint8
	b uint8
}

var (
	screenSize        = screenWidth * screenHeight
	pixels            = make([]byte, screenSize*4)
	firePixelsArray   = make([]byte, screenSize)
	fireColorsPalette = []palete{
		{r: 7, g: 7, b: 7},       //  0
		{r: 31, g: 7, b: 7},      //  1
		{r: 47, g: 15, b: 7},     //  2
		{r: 71, g: 15, b: 7},     //  3
		{r: 87, g: 23, b: 7},     //  4
		{r: 103, g: 31, b: 7},    //  5
		{r: 119, g: 31, b: 7},    //  6
		{r: 143, g: 39, b: 7},    //  7
		{r: 159, g: 47, b: 7},    //  8
		{r: 175, g: 63, b: 7},    //  9
		{r: 191, g: 71, b: 7},    // 10
		{r: 199, g: 71, b: 7},    // 11
		{r: 223, g: 79, b: 7},    // 12
		{r: 223, g: 87, b: 7},    // 13
		{r: 223, g: 87, b: 7},    // 14
		{r: 215, g: 95, b: 7},    // 15
		{r: 215, g: 95, b: 7},    // 16
		{r: 215, g: 103, b: 15},  // 17
		{r: 207, g: 111, b: 15},  // 18
		{r: 207, g: 119, b: 15},  // 19
		{r: 207, g: 127, b: 15},  // 20
		{r: 207, g: 135, b: 23},  // 21
		{r: 199, g: 135, b: 23},  // 22
		{r: 199, g: 143, b: 23},  // 23
		{r: 199, g: 151, b: 31},  // 24
		{r: 191, g: 159, b: 31},  // 25
		{r: 191, g: 159, b: 31},  // 26
		{r: 191, g: 167, b: 39},  // 27
		{r: 191, g: 167, b: 39},  // 28
		{r: 191, g: 175, b: 47},  // 29
		{r: 183, g: 175, b: 47},  // 30
		{r: 183, g: 183, b: 47},  // 31
		{r: 183, g: 183, b: 55},  // 32
		{r: 207, g: 207, b: 111}, // 33
		{r: 223, g: 223, b: 159}, // 34
		{r: 239, g: 239, b: 199}, // 35
		{r: 255, g: 255, b: 255}, // 36
	}
)

func createFireSource() {
	for i := screenSize - screenWidth; i < screenSize; i++ {
		firePixelsArray[i] = 36
	}
}

func calculateFirePropagation() {
	for column := 0; column < screenWidth; column++ {
		for row := 0; row < screenHeight; row++ {
			pixelIndex := column + (screenWidth * row)
			updateFireIntensityPerPixel(pixelIndex)
		}
	}
}

func updateFireIntensityPerPixel(currentPixelIndex int) {
	belowPixelIndex := currentPixelIndex + screenWidth
	if belowPixelIndex >= screenSize {
		return
	}

	decay := rand.Intn(3)
	belowPixelFireIntensity := int(firePixelsArray[belowPixelIndex])
	newFireIntensity := belowPixelFireIntensity - decay
	if newFireIntensity < 0 {
		newFireIntensity = 0
	}

	if currentPixelIndex-decay < 0 {
		return
	}
	firePixelsArray[currentPixelIndex-decay] = byte(newFireIntensity)
}

func renderFire() {
	for pos, v := range firePixelsArray {
		p := fireColorsPalette[v]
		pixels[pos*4] = p.r
		pixels[pos*4+1] = p.g
		pixels[pos*4+2] = p.b
		pixels[pos*4+3] = 0xff
	}
}

func update(screen *ebiten.Image) error {

	calculateFirePropagation()
	renderFire()

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	err := screen.ReplacePixels(pixels)
	return err
}

func main() {
	rand.Seed(time.Now().UnixNano())
	createFireSource()

	err := ebiten.Run(update, screenWidth, screenHeight, 6, "Fire")
	if err != nil {
		log.Fatal(err)
	}
}

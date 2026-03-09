package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Star struct {
	X, Y  float32
	Size  float32
	Phase float64
}

type Starfield struct {
	Stars []Star
}

func NewStarfield() *Starfield {
	sf := &Starfield{}

	horizontalFields := 8
	verticalFields := 5

	cellW := float32(ScreenWidth / horizontalFields)
	cellH := float32(ScreenHeight / verticalFields)

	for row := range verticalFields {
		for col := range horizontalFields {
			for i := 0; i < 2+rand.Intn(3); i++ {
				sf.Stars = append(sf.Stars, Star{
					X:     float32(col)*cellW + rand.Float32()*cellW,
					Y:     float32(row)*cellH + rand.Float32()*cellH,
					Size:  0.5 + rand.Float32()*1.5,
					Phase: rand.Float64() * math.Pi * 2,
				})
			}
		}
	}
	return sf
}

func (sf *Starfield) Draw(screen *ebiten.Image, tick int) {
	t := float64(tick) * 0.03
	for i := range sf.Stars {
		s := &sf.Stars[i]
		a := uint8(80 + math.Sin(t+s.Phase)*70)
		vector.FillCircle(screen, s.X, s.Y, s.Size, color.NRGBA{255, 255, 240, a}, true)
	}
}

package game

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	SpeedLineThreshold = 12.0
	SpeedLineMax       = 18.0
	SpeedLineLife      = 0.3
)

type SpeedLine struct {
	X, Y float64
	Life float64
}

type SpeedLines struct {
	Lines []SpeedLine
	Speed float64
	Len   float64
}

func (sl *SpeedLines) Update(vel Vec2) {
	dt := 1.0 / float64(ebiten.TPS())
	speed := vel.Len()

	if speed > SpeedLineThreshold {
		t := Clamp((speed-SpeedLineThreshold)/(SpeedLineMax-SpeedLineThreshold), 0, 1)

		if vel.Y > 0 {
			sl.Speed = -(200 + 300*t)
		} else {
			sl.Speed = 200 + 300*t
		}
		sl.Len = 25 + 50*t

		for range int(t*5) + 1 {
			sl.Lines = append(sl.Lines, SpeedLine{
				X:    rand.Float64() * float64(ScreenWidth),
				Y:    rand.Float64() * float64(ScreenHeight),
				Life: SpeedLineLife,
			})
		}
	}

	n := 0
	for i := range sl.Lines {
		l := &sl.Lines[i]
		l.Life -= dt
		if l.Life <= 0 {
			continue
		}
		l.Y += sl.Speed * dt
		sl.Lines[n] = *l
		n++
	}
	sl.Lines = sl.Lines[:n]
}

func (sl *SpeedLines) Draw(screen *ebiten.Image) {
	for i := range sl.Lines {
		l := &sl.Lines[i]
		alpha := uint8(l.Life / SpeedLineLife * 160)
		vector.StrokeLine(screen,
			float32(l.X), float32(l.Y),
			float32(l.X), float32(l.Y+sl.Len),
			1.5,
			color.NRGBA{255, 255, 255, alpha},
			true,
		)
	}
}

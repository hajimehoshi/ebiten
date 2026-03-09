package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	BellRadius      = 10.0
	BellSpawnChance = 0.35
	BellScoreValue  = 5
)

var bellImage *ebiten.Image

func loadBellAsset() {
	bellImage = loadImageFromFS("assets/textures/bell.png")
}

type Bell struct {
	Pos       Vec2
	Collected bool
	bobPhase  float64
}

func (b *Bell) Update() {
	b.bobPhase += 0.07
}

func (b *Bell) Draw(screen *ebiten.Image, cameraY, shakeX, shakeY float64) {
	if b.Collected {
		return
	}

	ox := float64(ScreenWidth)/2 + shakeX
	oy := float64(ScreenHeight)/2 - cameraY + shakeY

	dx := b.Pos.X + ox
	dy := b.Pos.Y + oy + math.Sin(b.bobPhase)*4

	op := &ebiten.DrawImageOptions{}
	iw := float64(bellImage.Bounds().Dx())
	ih := float64(bellImage.Bounds().Dy())
	op.GeoM.Translate(-iw/2, -ih/2)
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(bellImage, op)
}

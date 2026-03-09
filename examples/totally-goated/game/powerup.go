package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type PowerUpType int

const (
	PowerUpShield PowerUpType = iota
	PowerUpSuperDash
	PowerUpSlowFall
	PowerUpDoubleJump
)

const (
	PowerUpRadius      = 14.0
	PowerUpSpawnChance = 0.10
	SlowFallDuration   = 5.0
	SuperDashMult      = 1.5
	SlowFallGravityMul = 0.45
	PowerUpMinGap      = 3
)

var (
	puShieldImg    *ebiten.Image
	puSuperDashImg *ebiten.Image
	puSlowFallImg  *ebiten.Image
	puDoubleJmpImg *ebiten.Image
)

func loadPowerUpAssets() {
	puShieldImg = loadImageFromFS("assets/textures/powerup_shield.png")
	puSuperDashImg = loadImageFromFS("assets/textures/powerup_boostjump.png")
	puSlowFallImg = loadImageFromFS("assets/textures/powerup_slowfall.png")
	puDoubleJmpImg = loadImageFromFS("assets/textures/powerup_doublejump.png")
}

type PowerUp struct {
	Pos       Vec2
	Type      PowerUpType
	Collected bool
	bobPhase  float64
}

func (pu *PowerUp) Update() {
	pu.bobPhase += 0.05
}

func (pu *PowerUp) Draw(screen *ebiten.Image, cameraY, shakeX, shakeY float64) {
	if pu.Collected {
		return
	}
	ox := float64(ScreenWidth)/2 + shakeX
	oy := float64(ScreenHeight)/2 - cameraY + shakeY

	dx := pu.Pos.X + ox
	dy := pu.Pos.Y + oy + math.Sin(pu.bobPhase)*4

	var img *ebiten.Image
	switch pu.Type {
	case PowerUpShield:
		img = puShieldImg
	case PowerUpSuperDash:
		img = puSuperDashImg
	case PowerUpSlowFall:
		img = puSlowFallImg
	case PowerUpDoubleJump:
		img = puDoubleJmpImg
	}

	if img == nil {
		return
	}

	var glowClr color.NRGBA
	switch pu.Type {
	case PowerUpShield:
		glowClr = color.NRGBA{80, 160, 255, 50}
	case PowerUpSuperDash:
		glowClr = color.NRGBA{255, 100, 60, 50}
	case PowerUpSlowFall:
		glowClr = color.NRGBA{100, 230, 120, 50}
	case PowerUpDoubleJump:
		glowClr = color.NRGBA{200, 100, 255, 50}
	}
	glowR := float32(PowerUpRadius + 8 + math.Sin(pu.bobPhase*2)*3)
	vector.FillCircle(screen, float32(dx), float32(dy), glowR, glowClr, true)

	op := &ebiten.DrawImageOptions{}
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	op.GeoM.Translate(-iw/2, -ih/2)
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(img, op)
}

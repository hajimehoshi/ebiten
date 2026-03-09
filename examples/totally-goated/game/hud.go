package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func drawScaledText(screen *ebiten.Image, s string, x, y float64, scale float64, clr color.NRGBA) {
	w := len(s)*6 + 2
	h := 18

	tmp := ebiten.NewImage(w, h)
	ebitenutil.DebugPrintAt(tmp, s, 1, 1)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	screen.DrawImage(tmp, op)
}

func drawHUD(screen *ebiten.Image, g *Game) {
	meters := g.currentMeters()
	meterStr := fmt.Sprintf("%dm", meters)
	drawScaledText(screen, meterStr, 16, 12, 1.8, color.NRGBA{255, 255, 255, 220})

	if g.bellCount > 0 {
		bellStr := fmt.Sprintf("Bells: %d", g.bellCount)
		drawScaledText(screen, bellStr, 16, 38, 1.2, color.NRGBA{255, 210, 50, 220})
	}

	if g.comboCount > 1 {
		comboStr := fmt.Sprintf("x%d COMBO!", g.comboCount)
		pct := float32(g.comboTimer / comboWindow)
		a := uint8(220 * pct)
		drawScaledText(screen, comboStr, float64(ScreenWidth)/2-float64(len(comboStr))*6*0.8, 16, 1.6, color.NRGBA{255, 210, 50, a})
	}

	badgeX := float32(12)
	badgeY := float32(60)
	gap := float32(6)

	if g.Goat.HasShield {
		w := drawPowerUpBadge(screen, badgeX, badgeY, "SHIELD", -1,
			color.NRGBA{80, 160, 255, 200})
		badgeX += w + gap
	}
	if g.Goat.HasSuperDash {
		w := drawPowerUpBadge(screen, badgeX, badgeY, "SUPER DASH", -1,
			color.NRGBA{255, 100, 60, 200})
		badgeX += w + gap
	}
	if g.Goat.SlowFallTimer > 0 {
		w := drawPowerUpBadge(screen, badgeX, badgeY, "SLOW FALL", g.Goat.SlowFallTimer,
			color.NRGBA{100, 230, 120, 200})
		badgeX += w + gap
	}
	if g.Goat.HasDoubleJump {
		drawPowerUpBadge(screen, badgeX, badgeY, "DOUBLE JUMP", -1,
			color.NRGBA{200, 100, 255, 200})
	}
}

func drawPowerUpBadge(screen *ebiten.Image, x, y float32, label string, timer float64, clr color.NRGBA) float32 {
	text := label
	if timer > 0 {
		text = fmt.Sprintf("%s %.1fs", label, timer)
	}

	textW := float32(len(text)) * 6
	pad := float32(8)
	dotR := float32(4)
	dotGap := float32(6)
	badgeW := pad + dotR*2 + dotGap + textW + pad
	badgeH := float32(22)

	vector.FillRect(screen, x, y, badgeW, badgeH, color.NRGBA{0, 0, 0, 140}, true)
	vector.FillCircle(screen, x+pad+dotR, y+badgeH/2, dotR, clr, true)
	drawScaledText(screen, text,
		float64(x+pad+dotR*2+dotGap), float64(y+3), 1,
		color.NRGBA{255, 255, 255, 220})

	if timer > 0 {
		barX := x + pad
		barY := y + badgeH - 4
		barW := badgeW - pad*2
		pct := float32(timer / SlowFallDuration)
		if pct > 1 {
			pct = 1
		}
		vector.FillRect(screen, barX, barY, barW, 2, color.NRGBA{255, 255, 255, 30}, true)
		vector.FillRect(screen, barX, barY, barW*pct, 2, clr, true)
	}

	return badgeW
}

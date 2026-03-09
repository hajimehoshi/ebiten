package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type FloatingText struct {
	Pos     Vec2
	Text    string
	Life    float64
	MaxLife float64
	Color   color.NRGBA
	Scale   float64
}

func (g *Game) SpawnFloatingText(pos Vec2, text string, clr color.NRGBA) {
	g.floatingTexts = append(g.floatingTexts, FloatingText{
		Pos:     pos,
		Text:    text,
		Life:    0.8,
		MaxLife: 0.8,
		Color:   clr,
		Scale:   1.5,
	})
}

func (g *Game) updateFloatingTexts() {
	dt := 1.0 / float64(ebiten.TPS())
	n := 0
	for i := range g.floatingTexts {
		ft := &g.floatingTexts[i]
		ft.Life -= dt
		if ft.Life <= 0 {
			continue
		}
		ft.Pos.Y -= 40 * dt
		g.floatingTexts[n] = *ft
		n++
	}
	g.floatingTexts = g.floatingTexts[:n]
}

func (g *Game) drawFloatingTexts(screen *ebiten.Image, cameraY, shakeX, shakeY float64) {
	ox := float64(ScreenWidth)/2 + shakeX
	oy := float64(ScreenHeight)/2 - cameraY + shakeY

	for i := range g.floatingTexts {
		ft := &g.floatingTexts[i]
		t := ft.Life / ft.MaxLife
		a := uint8(float64(ft.Color.A) * t)
		clr := color.NRGBA{ft.Color.R, ft.Color.G, ft.Color.B, a}
		sx := ft.Pos.X + ox - float64(len(ft.Text))*3*ft.Scale
		sy := ft.Pos.Y + oy
		drawScaledText(screen, ft.Text, sx, sy, ft.Scale, clr)
	}
}

package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	_ "image/png"
)

type Game struct {
	level      *Level
	cameraY    float64
	Goat       Goat
	state      GameState
	menuPulse  float64
	tick       int
	menuGoat   *ebiten.Image
	score      float64
	bestScore  int
	bestMeters int
	bellCount  int
	bellScore  int
	bestBells  int
	background *ebiten.Image
	stars      *Starfield
	speedLines *SpeedLines
	shakeMag   float64
	shakeX     float64
	shakeY     float64

	floatingTexts []FloatingText
	flashAlpha    float64

	comboCount int
	comboTimer float64

	newBestScore  bool
	newBestMeters bool
	newBestBells  bool
}

type GameState int

const (
	GameMenu GameState = iota
	GamePlaying
	GameOver
)

func NewGame() *Game {
	img := loadImageFromFS("assets/textures/goat.png")
	background := loadImageFromFS("assets/textures/background.png")

	s := loadSave()
	return &Game{
		state:      GameMenu,
		menuGoat:   img,
		bestScore:  s.BestScore,
		bestMeters: s.BestMeters,
		bestBells:  s.BestBells,
		background: background,
		stars:      NewStarfield(),
		speedLines: &SpeedLines{},
	}
}

func (g *Game) startGame() {
	g.level = NewLevel()
	g.Goat = *NewGoat(0, 0)
	g.cameraY = 0
	g.score = 0
	g.bellCount = 0
	g.bellScore = 0
	g.state = GamePlaying
}

func (g *Game) Update() error {
	g.tick++
	g.menuPulse += 0.03

	switch g.state {
	case GameMenu:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.startGame()
		}

	case GamePlaying:
		g.Goat.Update(g.level, g.cameraY, g)
		g.level.Update()
		g.speedLines.Update(g.Goat.Vel)
		g.updateShake()
		g.updateFloatingTexts()
		g.updateCombo()
		if g.flashAlpha > 0 {
			g.flashAlpha -= 3.0 / float64(ebiten.TPS())
		}

		height := -g.Goat.Pos.Y
		if height > g.score {
			oldMeters := int(g.score / PixelsPerMeter)
			g.score = height
			newMeters := int(g.score / PixelsPerMeter)
			if newMeters/100 > oldMeters/100 {
				sfxMilestone.Play()
			}
		}

		targetY := g.Goat.Pos.Y
		if targetY < g.cameraY {
			speed := g.Goat.Vel.Len()
			lerpSpeed := 0.06 + Clamp(speed/30.0, 0, 0.12)
			g.cameraY += (targetY - g.cameraY) * lerpSpeed
		}

		deathY := g.cameraY + DeathMargin
		if g.Goat.Pos.Y > deathY {
			if g.Goat.HasShield {
				g.Goat.HasShield = false
				sfxShieldBreak.Play()

				g.Goat.Pos.Y = g.cameraY
				g.Goat.Pos.X = 0
				g.Goat.Vel = Vec2{0, 0}
				g.Goat.State = StateCharging
				g.Goat.ChargeTime = 0
				g.Goat.SlowFallTimer = 5.0
			} else {
				sfxDeath.Play()
				total := g.totalScore()
				meters := g.currentMeters()
				saveBest(total, meters, g.bellCount)
				g.newBestScore = total > g.bestScore
				g.newBestMeters = meters > g.bestMeters
				g.newBestBells = g.bellCount > g.bestBells
				if total > g.bestScore {
					g.bestScore = total
				}
				if meters > g.bestMeters {
					g.bestMeters = meters
				}
				if g.bellCount > g.bestBells {
					g.bestBells = g.bellCount
				}
				g.state = GameOver
			}
		}

		g.level.GenerateUntil(g.cameraY - 1000)

	case GameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.state = GameMenu
		}
	}

	return nil
}

func (g *Game) skyColor() color.RGBA {
	h := -g.cameraY / PixelsPerMeter
	t := Clamp(h/500.0, 0, 1)
	r := uint8(Lerp(30, 50, t))
	gr := uint8(Lerp(30, 20, t))
	b := uint8(Lerp(50, 70, t))
	return color.RGBA{R: r, G: gr, B: b, A: 255}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.state == GameMenu {
		screen.Fill(color.RGBA{R: 30, G: 30, B: 50, A: 255})
	} else {
		screen.Fill(g.skyColor())
	}
	g.stars.Draw(screen, g.tick)
	g.drawBackground(screen)

	switch g.state {
	case GameMenu:
		g.drawMenu(screen)
	case GamePlaying:
		g.drawGame(screen)
	case GameOver:
		g.drawGame(screen)
		g.drawGameOver(screen)
	}
}

func (g *Game) drawBackground(screen *ebiten.Image) {
	if g.background == nil {
		return
	}
	imgW := float64(g.background.Bounds().Dx())
	imgH := float64(g.background.Bounds().Dy())
	scale := float64(ScreenWidth) / imgW
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(0, float64(ScreenHeight)-imgH*scale)
	screen.DrawImage(g.background, op)
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	sw := float64(ScreenWidth)
	sh := float64(ScreenHeight)

	title := []string{
		` _____ ___ _____ _   _    _  __   __`,
		`|_   _/ _ \_   _/ \ | |  | | \ \ / /`,
		`  | || | | || |/ _ \| |  | |  \ V / `,
		`  | || |_| || / ___ \ |__| |__ | |  `,
		`  |_| \___/ |_/_/  \_\____|___)|_|  `,
		`                                     `,
		`   ____  ___    _  _____ _____ ____  `,
		`  / ___|/ _ \  / \|_   _| ____|  _ \ `,
		` | |  _| | | |/ _ \ | | |  _| | | | |`,
		` | |_| | |_| / ___ \| | | |___| |_| |`,
		`  \____|\___/_/   \_\_| |_____|____/ `,
	}

	titleY := int(sh * 0.05)
	for i, line := range title {
		ebitenutil.DebugPrintAt(screen, line, int(sw)/2-len(line)*3, titleY+i*16)
	}

	if g.menuGoat != nil {
		op := &ebiten.DrawImageOptions{}
		iw, ih := float64(g.menuGoat.Bounds().Dx()), float64(g.menuGoat.Bounds().Dy())
		scale := 3.0
		bob := math.Sin(g.menuPulse*2) * 6
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(sw/2-iw*scale/2, sh*0.42-ih*scale/2+bob)
		screen.DrawImage(g.menuGoat, op)
	}

	centerText := func(s string, y int) {
		ebitenutil.DebugPrintAt(screen, s, int(sw)/2-len(s)*3, y)
	}

	cy := int(sh * 0.6)
	centerText("CONTROLS:", cy)

	controls := [][2]string{
		{"CLICK + AIM", "Charge & aim your horn dash"},
		{"RELEASE", "Launch!"},
		{"A/D", "Air control while flying"},
		{"SPACE", "Double jump (if you have the item)"},
	}

	maxKeyW, maxDescW := 0, 0
	for _, c := range controls {
		if w := len(c[0]) * 6; w > maxKeyW {
			maxKeyW = w
		}
		if w := len(c[1]) * 6; w > maxDescW {
			maxDescW = w
		}
	}

	arrowW, gap := 18, 6
	blockX := int(sw)/2 - (maxKeyW+gap+arrowW+maxDescW)/2

	for i, c := range controls {
		y := cy + 20 + i*16
		ebitenutil.DebugPrintAt(screen, c[0], blockX+maxKeyW-len(c[0])*6, y)
		ebitenutil.DebugPrintAt(screen, "->", blockX+maxKeyW+gap, y)
		ebitenutil.DebugPrintAt(screen, c[1], blockX+maxKeyW+gap+arrowW, y)
	}

	centerText("Jakob Schwendinger & Jakob Wassertheurer", int(sh)-24)

	if g.bestScore > 0 {
		bestStr := fmt.Sprintf("Best Score: %d  |  %dm  |  %d bells", g.bestScore, g.bestMeters, g.bestBells)
		bw := float64(len(bestStr)) * 6
		drawScaledText(screen, bestStr, sw/2-bw/2, sh*0.82, 1.0, color.NRGBA{255, 210, 50, 180})
	}

	if g.tick%60 < 40 {
		prompt := "Click or Space to start"
		pw := float64(len(prompt)) * 6
		drawScaledText(screen, prompt, sw/2-pw*1.2/2, sh*0.88, 1.2, color.NRGBA{255, 255, 255, 200})
	}
}

func (g *Game) AddShake(amount float64) {
	g.shakeMag = math.Min(g.shakeMag+amount, 16.0)
}

func (g *Game) updateShake() {
	if g.shakeMag > 0.2 {
		g.shakeX = math.Sin(float64(g.tick)*1.1)*g.shakeMag + math.Cos(float64(g.tick)*2.3)*g.shakeMag*0.5
		g.shakeY = math.Cos(float64(g.tick)*1.7)*g.shakeMag + math.Sin(float64(g.tick)*3.1)*g.shakeMag*0.5
		g.shakeMag *= 0.88
	} else {
		g.shakeX = 0
		g.shakeY = 0
		g.shakeMag = 0
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {
	g.level.Draw(screen, g.cameraY, g.shakeX, g.shakeY)
	g.Goat.Draw(screen, g.cameraY, g.shakeX, g.shakeY)
	g.drawFloatingTexts(screen, g.cameraY, g.shakeX, g.shakeY)
	g.speedLines.Draw(screen)

	drawHUD(screen, g)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	white := color.NRGBA{255, 255, 255, 220}
	gold := color.NRGBA{255, 210, 50, 220}
	dim := color.NRGBA{180, 180, 180, 180}

	cx := float64(ScreenWidth) / 2
	cy := float64(ScreenHeight)/2 - 20

	title := "GAME OVER"
	drawScaledText(screen, title, cx-float64(len(title))*6*1.5/2, cy-90, 1.5, color.NRGBA{255, 80, 80, 240})

	yOff := 0.0
	if g.newBestScore || g.newBestMeters || g.newBestBells {
		hsText := "NEW HIGHSCORE!"
		hsW := float64(len(hsText)*8 + 2)
		drawScaledText(screen, hsText, cx-hsW/2, cy-62, 1.5, color.NRGBA{255, 210, 50, 255})
		yOff = 20
	}

	vector.FillRect(screen, float32(cx-100), float32(cy-55+yOff), 200, 1, color.NRGBA{255, 255, 255, 60}, true)

	heightStr := fmt.Sprintf("%d m", g.currentMeters())
	bellsStr := fmt.Sprintf("%d bells", g.bellCount)

	drawScaledText(screen, "Height:", cx-100, cy-40+yOff, 1.0, dim)
	drawScaledText(screen, heightStr, cx+40, cy-40+yOff, 1.0, white)

	drawScaledText(screen, "Bells:", cx-100, cy-20+yOff, 1.0, dim)
	drawScaledText(screen, bellsStr, cx+40, cy-20+yOff, 1.0, gold)

	vector.FillRect(screen, float32(cx-100), float32(cy+1+yOff), 200, 1, color.NRGBA{255, 255, 255, 60}, true)

	totalStr := fmt.Sprintf("%d", g.totalScore())
	drawScaledText(screen, "SCORE:", cx-100, cy+14+yOff, 1.3, white)
	drawScaledText(screen, totalStr, cx+40, cy+14+yOff, 1.3, white)

	vector.FillRect(screen, float32(cx-100), float32(cy+42+yOff), 200, 1, color.NRGBA{255, 255, 255, 40}, true)

	bestScoreStr := fmt.Sprintf("%d", g.bestScore)
	bestMetersStr := fmt.Sprintf("%d m", g.bestMeters)
	bestBellsStr := fmt.Sprintf("%d", g.bestBells)

	newBestClr := color.NRGBA{255, 210, 50, 255}

	drawScaledText(screen, "BEST", cx-100, cy+52+yOff, 1.0, dim)
	scoreClr := dim
	if g.newBestScore {
		scoreClr = newBestClr
	}
	drawScaledText(screen, "Score:", cx-100, cy+70+yOff, 1.0, dim)
	drawScaledText(screen, bestScoreStr, cx+40, cy+70+yOff, 1.0, scoreClr)

	metersClr := dim
	if g.newBestMeters {
		metersClr = newBestClr
	}
	drawScaledText(screen, "Height:", cx-100, cy+86+yOff, 1.0, dim)
	drawScaledText(screen, bestMetersStr, cx+40, cy+86+yOff, 1.0, metersClr)

	bellsClr := gold
	if g.newBestBells {
		bellsClr = newBestClr
	}
	drawScaledText(screen, "Bells:", cx-100, cy+102+yOff, 1.0, dim)
	drawScaledText(screen, bestBellsStr, cx+40, cy+102+yOff, 1.0, bellsClr)

	if g.tick%60 < 40 {
		prompt := "Click or Space to continue"
		pw := float64(len(prompt)) * 6
		drawScaledText(screen, prompt, cx-pw/2, cy+130+yOff, 1.0, dim)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}

func toMeters(px float64) int {
	return int(px / PixelsPerMeter)
}

func (g *Game) currentMeters() int {
	return toMeters(g.score)
}

func (g *Game) totalScore() int {
	return g.currentMeters() + g.bellScore
}

func (g *Game) updateCombo() {
	if g.comboTimer > 0 {
		g.comboTimer -= 1.0 / float64(ebiten.TPS())
		if g.comboTimer <= 0 {
			g.comboCount = 0
		}
	}
}

func (g *Game) OnBellCollected(pos Vec2) {
	g.bellCount++
	g.comboCount++
	g.comboTimer = comboWindow

	mult := g.comboCount
	if mult > 5 {
		mult = 5
	}
	points := BellScoreValue * mult
	g.bellScore += points

	text := fmt.Sprintf("+%d", points)
	if g.comboCount > 1 {
		text = fmt.Sprintf("+%d x%d", points, mult)
	}
	g.SpawnFloatingText(pos, text, color.NRGBA{255, 210, 50, 255})
	g.Goat.Particles.SpawnDust(pos, 6)
}

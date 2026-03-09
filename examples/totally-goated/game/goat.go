package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GoatState int

const (
	StateAir GoatState = iota
	StateWall
	StateCharging
)

type WallSide int

const (
	WallNone WallSide = iota
	WallLeft
	WallRight
)

type Goat struct {
	Pos          Vec2
	Vel          Vec2
	State        GoatState
	Wall         WallSide
	WallPlatIdx  int
	ClingTimer   int
	FacingDir    float64
	Image        *ebiten.Image
	DashSpeedMod float64

	ChargeTime        float64
	DashCooldownTimer float64
	LastDashPlatIdx   int
	Particles         ParticleSystem

	HasShield     bool
	HasSuperDash  bool
	SlowFallTimer float64
	HasDoubleJump bool

	SquashX  float64
	SquashY  float64
	Rotation float64
}

func NewGoat(x, y float64) *Goat {
	goatImage := loadImageFromFS("assets/textures/goat.png")
	return &Goat{
		Pos:             Vec2{x, y},
		FacingDir:       1,
		WallPlatIdx:     -1,
		LastDashPlatIdx: -1,
		Image:           goatImage,
		DashSpeedMod:    1.0,
		SquashX:         1.0,
		SquashY:         1.0,
	}
}

func (g *Goat) Update(level *Level, cameraY float64, game *Game) {
	g.SquashX = Lerp(g.SquashX, 1.0, 0.12)
	g.SquashY = Lerp(g.SquashY, 1.0, 0.12)
	g.Rotation = Lerp(g.Rotation, 0.0, 0.08)

	switch g.State {
	case StateAir:
		g.updateAir(level, game)
	case StateWall:
		g.updateWall(level)
	case StateCharging:
		g.updateCharging(level, cameraY, game)
	}

	g.Particles.Update()

	if g.State == StateAir {
		g.Particles.Emit(g.Pos, g.Vel)
		if g.Vel.Len() > 2 {
			targetRot := math.Atan2(g.Vel.Y, g.Vel.X*g.FacingDir) * 0.15
			g.Rotation = Lerp(g.Rotation, targetRot, 0.05)
		}
	}
}

func (g *Goat) updateAir(level *Level, game *Game) {
	grav := Gravity
	if g.SlowFallTimer > 0 {
		if g.Vel.Y > 0 {
			grav *= SlowFallGravityMul
		}
		g.SlowFallTimer -= 1.0 / float64(ebiten.TPS())
	}

	g.Vel.Y += grav
	if g.Vel.Y > MaxFallSpeed {
		g.Vel.Y = MaxFallSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.Vel.X -= AirControlAccel
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.Vel.X += AirControlAccel
	}

	g.Vel.X *= DashDrag

	if g.HasDoubleJump && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Vel.Y = -DoubleJumpSpeed
		g.HasDoubleJump = false
		g.SquashX = 0.8
		g.SquashY = 1.25
		g.Particles.SpawnDust(g.Pos, 8)
		sfxPowerup.Play()
	}

	g.Pos = g.Pos.Add(g.Vel)
	g.collectPowerUps(level, game)
	g.collectBells(level, game)
	g.resolveCollisions(level, game)
}

func (g *Goat) collectPowerUps(level *Level, game *Game) {
	gw := float64(g.Image.Bounds().Dx())
	gh := float64(g.Image.Bounds().Dy())

	for i := range level.PowerUps {
		pu := &level.PowerUps[i]
		if pu.Collected {
			continue
		}

		dx := pu.Pos.X - Clamp(pu.Pos.X, g.Pos.X-gw/2, g.Pos.X+gw/2)
		dy := pu.Pos.Y - Clamp(pu.Pos.Y, g.Pos.Y-gh/2, g.Pos.Y+gh/2)
		if dx*dx+dy*dy < PowerUpRadius*PowerUpRadius {
			pu.Collected = true
			sfxPowerup.Play()

			var name string
			switch pu.Type {
			case PowerUpShield:
				g.HasShield = true
				name = "SHIELD"
			case PowerUpSuperDash:
				g.HasSuperDash = true
				name = "SUPER DASH"
			case PowerUpSlowFall:
				g.SlowFallTimer = SlowFallDuration
				name = "SLOW FALL"
			case PowerUpDoubleJump:
				g.HasDoubleJump = true
				name = "DOUBLE JUMP"
			}

			if game != nil {
				game.flashAlpha = 0.3
				game.SpawnFloatingText(pu.Pos, name, color.NRGBA{255, 255, 255, 255})
				game.AddShake(3)
			}
			g.Particles.SpawnImpactRing(pu.Pos)
		}
	}
}

func (g *Goat) collectBells(level *Level, game *Game) {
	gw := float64(g.Image.Bounds().Dx())
	gh := float64(g.Image.Bounds().Dy())

	for i := range level.Bells {
		b := &level.Bells[i]
		if b.Collected {
			continue
		}

		dx := b.Pos.X - Clamp(b.Pos.X, g.Pos.X-gw/2, g.Pos.X+gw/2)
		dy := b.Pos.Y - Clamp(b.Pos.Y, g.Pos.Y-gh/2, g.Pos.Y+gh/2)
		if dx*dx+dy*dy < BellRadius*BellRadius {
			b.Collected = true
			sfxBellPickup.Play()
			if game != nil {
				game.OnBellCollected(b.Pos)
			}
		}
	}
}

func (g *Goat) updateWall(level *Level) {
	if g.WallPlatIdx >= 0 && g.WallPlatIdx < len(level.Platforms) {
		if level.Platforms[g.WallPlatIdx].Destroyed {
			g.detachFromWall()
			return
		}
	}
	if g.DashCooldownTimer > 0 {
		g.DashCooldownTimer -= 1.0 / float64(ebiten.TPS())
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.DashCooldownTimer <= 0 {
		g.State = StateCharging
		g.ChargeTime = 0
		return
	}

	p := &level.Platforms[g.WallPlatIdx]

	slideSpeed := WallSlideSpeed
	if p.Type == PlatIce {
		slideSpeed = IceSlideSpeed
	}

	g.Vel.Y = slideSpeed
	g.Vel.X = 0
	g.Pos.Y += g.Vel.Y
	g.ClingTimer++

	g.checkStillOnWall(level)
}

func (g *Goat) updateCharging(level *Level, cameraY float64, game *Game) {
	if g.WallPlatIdx >= 0 && g.WallPlatIdx < len(level.Platforms) {
		if level.Platforms[g.WallPlatIdx].Destroyed {
			g.detachFromWall()
			return
		}
	}
	dt := 1.0 / float64(ebiten.TPS())
	g.ChargeTime += dt
	if g.ChargeTime > FullChargeTime {
		g.ChargeTime = FullChargeTime
	}

	slideSpeed := WallSlideSpeed
	if g.WallPlatIdx >= 0 && g.WallPlatIdx < len(level.Platforms) {
		if level.Platforms[g.WallPlatIdx].Type == PlatIce {
			slideSpeed = IceSlideSpeed
		}
	}
	g.Pos.Y += slideSpeed
	g.ClingTimer++

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		dir := g.aimDir(cameraY)

		speed := Lerp(DashMinSpeed, DashMaxSpeed, g.ChargingPercentage())
		speed *= g.DashSpeedMod
		if g.HasSuperDash {
			speed *= SuperDashMult
			g.HasSuperDash = false
		}
		pct := g.ChargingPercentage()
		switch {
		case pct > 0.8:
			sfxDashPower15.Play()
		case pct > 0.4:
			sfxDashPower10.Play()
		default:
			sfxDashPower5.Play()
		}
		sfxChargeRelease.Play()

		g.Vel = dir.Scale(speed)
		g.State = StateAir
		g.Wall = WallNone
		g.LastDashPlatIdx = g.WallPlatIdx
		g.WallPlatIdx = -1
		g.DashCooldownTimer = DashCooldown

		g.SquashX = 0.75
		g.SquashY = 1.2
		g.Rotation = AngleFromDir(dir) * 0.5
		g.Particles.SpawnDashSparks(g.Pos, dir, 12)
		if game != nil {
			game.AddShake(speed * 0.2)
		}
		return
	}

	g.checkStillOnWall(level)
}

func (g *Goat) checkStillOnWall(level *Level) {
	gh := float64(g.Image.Bounds().Dy())
	if g.WallPlatIdx >= 0 && g.WallPlatIdx < len(level.Platforms) {
		p := &level.Platforms[g.WallPlatIdx]
		stillOn := g.Pos.Y+gh/2 >= p.Y && g.Pos.Y-gh/2 <= p.Y+p.H
		if !stillOn {
			g.detachFromWall()
		}
	} else {
		g.detachFromWall()
	}
}

func (g *Goat) resolveCollisions(level *Level, game *Game) {
	gw := float64(g.Image.Bounds().Dx())
	gh := float64(g.Image.Bounds().Dy())

	for idx := range level.Platforms {
		p := &level.Platforms[idx]

		if p.Destroyed {
			continue
		}

		goatL := g.Pos.X - gw/2
		goatR := g.Pos.X + gw/2
		goatT := g.Pos.Y - gh/2
		goatB := g.Pos.Y + gh/2

		if goatR <= p.X || goatL >= p.X+p.W || goatB <= p.Y || goatT >= p.Y+p.H {
			continue
		}

		penL := goatR - p.X
		penR := (p.X + p.W) - goatL
		penT := goatB - p.Y
		penB := (p.Y + p.H) - goatT

		minPen := math.Min(math.Min(penL, penR), math.Min(penT, penB))

		switch {
		case minPen == penT && g.Vel.Y >= 0:
			g.Pos.Y = p.Y - gh/2

			impact := g.Vel.Len()
			if impact > 3 {
				g.Particles.SpawnLandImpact(Vec2{g.Pos.X, p.Y}, impact)
				g.SquashX = 1.3
				g.SquashY = 0.7
			}

			distL := g.Pos.X - p.X
			distR := (p.X + p.W) - g.Pos.X
			if distL < distR {
				g.Pos.X = p.X - gw/2
				g.Pos.Y = p.Y + gh/2
				g.attachToWall(WallRight, idx, level, game)
			} else {
				g.Pos.X = p.X + p.W + gw/2
				g.Pos.Y = p.Y + gh/2
				g.attachToWall(WallLeft, idx, level, game)
			}
			return

		case minPen == penL:
			g.Pos.X = p.X - gw/2
			g.attachToWall(WallRight, idx, level, game)
			return

		case minPen == penR:
			g.Pos.X = p.X + p.W + gw/2
			g.attachToWall(WallLeft, idx, level, game)
			return

		case minPen == penB:
			g.Pos.Y = p.Y + p.H + gh/2
			g.Vel.Y = math.Abs(g.Vel.Y) * 0.15
		}
	}
}

func (g *Goat) attachToWall(side WallSide, platIdx int, level *Level, game *Game) {
	p := &level.Platforms[platIdx]

	wasMoving := g.Vel.Len() > 2
	impactSpeed := g.Vel.Len()

	if p.Type == PlatBouncy {
		sfxBounce.Play()
		if side == WallLeft || side == WallRight {
			extra := math.Abs(g.Vel.Y) * BouncySideAbsorb
			g.Vel.X = -g.Vel.X * BouncyReflect
			if g.Vel.X < 0 {
				g.Vel.X -= extra
			} else {
				g.Vel.X += extra
			}
			if math.Abs(g.Vel.X) < BouncyMinSpeed {
				if g.Vel.X < 0 {
					g.Vel.X = -BouncyMinSpeed
				} else {
					g.Vel.X = BouncyMinSpeed
				}
			}

			g.Vel.Y *= 0.3
		} else if g.Vel.Y > 0 {
			g.Vel.Y = -g.Vel.Y * BouncyReflect
			if g.Vel.Y > -BouncyMinSpeed {
				g.Vel.Y = -BouncyMinSpeed
			}
		}

		g.State = StateAir
		g.Wall = WallNone
		g.WallPlatIdx = -1
		return
	}

	switch p.Type {
	case PlatIce:
		sfxIce.Play()
	case PlatCrumbly:
		if !p.CrumbleStarted {
			p.CrumbleStarted = true
		}
		sfxWallHit.Play()
	default:
		sfxWallHit.Play()
	}

	g.DashSpeedMod = 1.0
	if p.Type == PlatSticky {
		g.DashSpeedMod = StickyDashMult
	}

	g.State = StateWall
	g.Wall = side
	g.WallPlatIdx = platIdx
	g.Vel = Vec2{}
	g.ClingTimer = 0
	if platIdx != g.LastDashPlatIdx {
		g.DashCooldownTimer = 0
	}

	if side == WallLeft {
		g.FacingDir = -1
	} else {
		g.FacingDir = 1
	}

	if wasMoving {
		g.SquashX = 1.4
		g.SquashY = 0.7
		g.Particles.SpawnDust(g.Pos, 6)
		g.Particles.SpawnImpactRing(g.Pos)
		if game != nil && impactSpeed > 4 {
			game.AddShake(impactSpeed * 0.3)
		}
	}
}

func (g *Goat) detachFromWall() {
	g.State = StateAir
	g.Wall = WallNone
	g.WallPlatIdx = -1
}

func (g *Goat) aimDir(cameraY float64) Vec2 {
	mx, my := ebiten.CursorPosition()
	worldX := float64(mx) - float64(ScreenWidth)/2
	worldY := float64(my) - float64(ScreenHeight)/2 + cameraY
	dir := (Vec2{worldX, worldY}).Sub(g.Pos).Normalize()
	if dir.Len() < 0.0001 {
		if g.Wall == WallLeft {
			dir = Vec2{1, -0.35}.Normalize()
		} else {
			dir = Vec2{-1, -0.35}.Normalize()
		}
	}
	return dir
}

func (g *Goat) Draw(screen *ebiten.Image, cameraY, shakeX, shakeY float64) {
	g.Particles.Draw(screen, cameraY, shakeX, shakeY)

	offsetX := float64(ScreenWidth)/2 + shakeX
	offsetY := float64(ScreenHeight)/2 - cameraY + shakeY

	if g.HasShield {
		cx := float32(g.Pos.X + offsetX)
		cy := float32(g.Pos.Y + offsetY)
		vector.StrokeCircle(screen, cx, cy, 22, 2, color.NRGBA{80, 180, 255, 160}, true)
	}

	if g.SlowFallTimer > 0 {
		cx := float32(g.Pos.X + offsetX)
		cy := float32(g.Pos.Y + offsetY)
		vector.FillCircle(screen, cx, cy, 18, color.NRGBA{100, 230, 120, 40}, true)
	}

	if g.State == StateCharging {
		g.drawDashAimLine(screen, offsetX, offsetY, cameraY)
		g.drawDashTrajectory(screen, offsetX, offsetY, cameraY)
		g.drawChargeCircle(screen, offsetX, offsetY)
		g.drawChargeBar(screen, offsetX, offsetY)
	}

	op := &ebiten.DrawImageOptions{}
	iw := float64(g.Image.Bounds().Dx())
	ih := float64(g.Image.Bounds().Dy())

	op.GeoM.Translate(-iw/2, -ih/2)
	if g.FacingDir < 0 {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Scale(g.SquashX, g.SquashY)
	op.GeoM.Rotate(g.Rotation)
	op.GeoM.Translate(g.Pos.X+offsetX, g.Pos.Y+offsetY)
	screen.DrawImage(g.Image, op)
}

func (g *Goat) ChargingPercentage() float64 {
	return Clamp(g.ChargeTime/FullChargeTime, 0, 1)
}

func (g *Goat) drawChargeBar(screen *ebiten.Image, offsetX, offsetY float64) {
	pct := g.ChargingPercentage()

	barW := float32(60)
	barH := float32(5)
	barX := float32(g.Pos.X+offsetX) - barW/2
	barY := float32(g.Pos.Y+offsetY) + float32(g.Image.Bounds().Dy())/2 + 10

	vector.FillRect(screen, barX, barY, barW, barH, color.RGBA{50, 50, 50, 200}, false)

	fillColor := chargingColor(pct)
	vector.FillRect(screen, barX, barY, barW*float32(pct), barH, fillColor, false)
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, color.RGBA{255, 255, 255, 255}, false)
}

func (g *Goat) drawChargeCircle(screen *ebiten.Image, offsetX, offsetY float64) {
	pct := g.ChargingPercentage()

	cx := float32(g.Pos.X + offsetX)
	cy := float32(g.Pos.Y + offsetY)

	fillColor := chargingColor(pct)
	fillColor.A = 100

	bounds := g.Image.Bounds()
	maxDimension := max(bounds.Dx(), bounds.Dy())
	radius := Lerp(float64(maxDimension)*.8, float64(maxDimension)*1.5, pct)

	vector.FillCircle(screen, cx, cy, float32(radius), fillColor, true)
}

func chargingColor(pct float64) color.NRGBA {
	r := uint8(pct * 255)
	g := uint8((1 - pct) * 255)
	return color.NRGBA{r, g, 0, 255}
}

func (g *Goat) drawDashAimLine(screen *ebiten.Image, offsetX, offsetY, cameraY float64) {
	dir := g.aimDir(cameraY)

	power := g.ChargingPercentage()
	lineLen := 40.0 + power*70.0

	sx := g.Pos.X + offsetX
	sy := g.Pos.Y + offsetY
	endX := sx + dir.X*lineLen
	endY := sy + dir.Y*lineLen

	for i := range 8 {
		t1 := float64(i) / 8.0
		t2 := (float64(i) + 0.5) / 8.0
		x1 := sx + (endX-sx)*t1
		y1 := sy + (endY-sy)*t1
		x2 := sx + (endX-sx)*t2
		y2 := sy + (endY-sy)*t2

		r := uint8(255)
		gc := uint8(Lerp(255, 80, power))
		b := uint8(Lerp(200, 30, power))

		vector.StrokeLine(
			screen,
			float32(x1), float32(y1),
			float32(x2), float32(y2),
			float32(2+power*2),
			color.RGBA{r, gc, b, 200},
			true,
		)
	}

	arrowSz := 6.0 + power*5.0
	perp := Vec2{-dir.Y, dir.X}
	ax1x := endX + (-dir.X+perp.X*0.5)*arrowSz
	ax1y := endY + (-dir.Y+perp.Y*0.5)*arrowSz
	ax2x := endX + (-dir.X-perp.X*0.5)*arrowSz
	ax2y := endY + (-dir.Y-perp.Y*0.5)*arrowSz

	vector.StrokeLine(screen, float32(endX), float32(endY), float32(ax1x), float32(ax1y),
		float32(2+power*2), color.RGBA{255, 200, 50, 255}, true)
	vector.StrokeLine(screen, float32(endX), float32(endY), float32(ax2x), float32(ax2y),
		float32(2+power*2), color.RGBA{255, 200, 50, 255}, true)
}

func (g *Goat) drawDashTrajectory(screen *ebiten.Image, offsetX, offsetY, cameraY float64) {
	dir := g.aimDir(cameraY)

	speed := Lerp(DashMinSpeed, DashMaxSpeed, g.ChargingPercentage())
	speed *= g.DashSpeedMod

	if g.HasSuperDash {
		speed *= SuperDashMult
	}

	vel := dir.Scale(speed)
	pos := g.Pos

	for i := range 40 {
		vel.Y += Gravity
		if vel.Y > MaxFallSpeed {
			vel.Y = MaxFallSpeed
		}
		vel.X *= DashDrag
		pos = pos.Add(vel)

		if i%3 == 0 {
			sx := pos.X + offsetX
			sy := pos.Y + offsetY
			alpha := uint8(Clamp(float64(120-i*3), 20, 120))
			sz := float32(Clamp(3.0-float64(i)*0.05, 1, 3))
			vector.FillCircle(screen, float32(sx), float32(sy), sz, color.RGBA{255, 255, 255, alpha}, true)
		}
	}
}

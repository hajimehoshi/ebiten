package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Particle struct {
	Pos     Vec2
	Vel     Vec2
	Life    float64
	MaxLife float64
	Size    float64
	Color   color.NRGBA
	Ring    bool
	Gravity float64
	Drag    float64
}

type ParticleSystem struct {
	Particles []Particle
}

func (ps *ParticleSystem) Emit(pos, vel Vec2) {
	if vel.Len() < 3 || rand.Float64() > 0.4 {
		return
	}
	ps.Particles = append(ps.Particles, Particle{
		Pos: Vec2{
			pos.X + (rand.Float64()-0.5)*6,
			pos.Y + (rand.Float64()-0.5)*6,
		},
		Vel:     Vec2{(rand.Float64() - 0.5) * 0.5, -rand.Float64() * 0.3},
		Life:    0.4 + rand.Float64()*0.3,
		MaxLife: 0.7,
		Size:    2,
		Color:   color.NRGBA{220, 200, 170, 255},
		Drag:    0.02,
	})
}

func (ps *ParticleSystem) SpawnDust(pos Vec2, count int) {
	for i := 0; i < count; i++ {
		angle := rand.Float64() * math.Pi * 2
		speed := 0.5 + rand.Float64()*2.0
		ps.Particles = append(ps.Particles, Particle{
			Pos:     pos,
			Vel:     Vec2{math.Cos(angle) * speed, math.Sin(angle) * speed},
			Life:    0.3 + rand.Float64()*0.3,
			MaxLife: 0.6,
			Size:    2 + rand.Float64()*3,
			Color:   color.NRGBA{200, 180, 150, 255},
			Gravity: 0.05,
			Drag:    0.02,
		})
	}
}

func (ps *ParticleSystem) SpawnDashSparks(pos Vec2, dir Vec2, count int) {
	baseAngle := AngleFromDir(dir)
	for i := 0; i < count; i++ {
		spread := (rand.Float64() - 0.5) * 1.2
		angle := baseAngle + math.Pi + spread
		speed := 2 + rand.Float64()*4.0
		ps.Particles = append(ps.Particles, Particle{
			Pos:     pos,
			Vel:     Vec2{math.Cos(angle) * speed, math.Sin(angle) * speed},
			Life:    0.2 + rand.Float64()*0.3,
			MaxLife: 0.5,
			Size:    2 + rand.Float64()*3,
			Color:   color.NRGBA{255, 200, 50, 255},
			Drag:    0.01,
		})
	}
}

func (ps *ParticleSystem) SpawnImpactRing(pos Vec2) {
	ps.Particles = append(ps.Particles, Particle{
		Pos:     pos,
		Life:    0.3,
		MaxLife: 0.3,
		Size:    4,
		Color:   color.NRGBA{255, 255, 255, 180},
		Ring:    true,
	})
}

func (ps *ParticleSystem) SpawnLandImpact(pos Vec2, power float64) {
	count := int(power * 2)
	if count > 12 {
		count = 12
	}
	for i := 0; i < count; i++ {
		angle := -math.Pi + rand.Float64()*math.Pi
		speed := 1 + rand.Float64()*power*0.4
		ps.Particles = append(ps.Particles, Particle{
			Pos:     pos,
			Vel:     Vec2{math.Cos(angle) * speed, math.Sin(angle) * speed},
			Life:    0.2 + rand.Float64()*0.25,
			MaxLife: 0.45,
			Size:    2 + rand.Float64()*3,
			Color:   color.NRGBA{180, 160, 130, 255},
			Gravity: 0.1,
			Drag:    0.03,
		})
	}
}

func (ps *ParticleSystem) Update() {
	dt := 1.0 / float64(ebiten.TPS())
	n := 0
	for i := range ps.Particles {
		p := &ps.Particles[i]
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.Vel.Y += p.Gravity
		p.Vel = p.Vel.Scale(1.0 - p.Drag)
		p.Pos = p.Pos.Add(p.Vel)
		ps.Particles[n] = *p
		n++
	}
	ps.Particles = ps.Particles[:n]
}

func (ps *ParticleSystem) Draw(screen *ebiten.Image, cameraY, shakeX, shakeY float64) {
	ox := float64(ScreenWidth)/2 + shakeX
	oy := float64(ScreenHeight)/2 - cameraY + shakeY

	for i := range ps.Particles {
		p := &ps.Particles[i]
		t := p.Life / p.MaxLife
		if t < 0 {
			t = 0
		}
		sx := float32(p.Pos.X + ox)
		sy := float32(p.Pos.Y + oy)
		a := uint8(float64(p.Color.A) * t)
		c := color.NRGBA{p.Color.R, p.Color.G, p.Color.B, a}

		if p.Ring {
			vector.StrokeCircle(screen, sx, sy, float32(p.Size+(1-t)*30), 2, c, true)
		} else {
			sz := float32(p.Size * t)
			if sz < 0.5 {
				sz = 0.5
			}
			vector.FillCircle(screen, sx, sy, sz, c, true)
		}
	}
}

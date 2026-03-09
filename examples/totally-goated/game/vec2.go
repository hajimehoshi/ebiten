package game

import "math"

type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2      { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Len() float64         { return math.Sqrt(v.X*v.X + v.Y*v.Y) }

func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l < 0.0001 {
		return Vec2{}
	}
	return Vec2{v.X / l, v.Y / l}
}

func (v Vec2) Lerp(o Vec2, t float64) Vec2 {
	return Vec2{v.X + (o.X-v.X)*t, v.Y + (o.Y-v.Y)*t}
}

func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func AngleFromDir(dir Vec2) float64 {
	return math.Atan2(dir.Y, dir.X)
}

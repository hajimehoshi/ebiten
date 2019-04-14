// Copyright 2019 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package math

import (
	"math"
)

type Point struct {
	X float32
	Y float32
}

type Vec2 struct {
	X float32
	Y float32
}

func (p Vec2) Cross(other Vec2) float32 {
	return p.X*other.Y - p.Y*other.X
}

type Segment struct {
	P0 Point
	P1 Point
}

func (s Segment) Translate(offset float32) Segment {
	a := math.Atan2(float64(s.P1.Y-s.P0.Y), float64(s.P1.X-s.P0.X))
	si, co := math.Sincos(a + math.Pi/2)
	dx, dy := float32(co)*offset, float32(si)*offset
	return Segment{
		P0: Point{s.P0.X + dx, s.P0.Y + dy},
		P1: Point{s.P1.X + dx, s.P1.Y + dy},
	}
}

func (s Segment) IntersectionAsLines(other Segment) Point {
	v1 := Vec2{other.P1.X - other.P0.X, other.P1.Y - other.P0.Y}
	d0 := v1.Cross(Vec2{s.P0.X - other.P0.X, s.P0.Y - other.P0.Y})
	d1 := -v1.Cross(Vec2{s.P1.X - other.P1.X, s.P1.Y - other.P1.Y})
	t := d0 / (d0 + d1)
	return Point{s.P0.X + (s.P1.X-s.P0.X)*t, s.P0.Y + (s.P1.Y-s.P0.Y)*t}
}

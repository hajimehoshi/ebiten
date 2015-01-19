// Copyright 2015 Hajime Hoshi
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

package shape

import (
	"github.com/hajimehoshi/ebiten"
	"image/color"
	"math"
)

func DrawEllipse(s *ebiten.Image, x, y, width, height int, clr color.Color) error {
	return s.DrawLines(&ellipsesLines{&rect{x, y, width, height, clr}, 0, 2 * math.Pi})
}

func DrawEllipses(s *ebiten.Image, rects ebiten.Rects) error {
	return s.DrawLines(&ellipsesLines{rects, 0, 2 * math.Pi})
}

func DrawArc(s *ebiten.Image, x, y, width, height int, angle0, angle1 float64, clr color.Color) error {
	return s.DrawLines(&ellipsesLines{&rect{x, y, width, height, clr}, angle0, angle1})
}

type ellipsesLines struct {
	ebiten.Rects
	angle0, angle1 float64
}

func (e *ellipsesLines) lineNum() int {
	return 64
}

func (e *ellipsesLines) Len() int {
	return e.Rects.Len() * e.lineNum()
}

func round(x float64) int {
	return int(x + 0.5)
}

func (e *ellipsesLines) Points(i int) (x0, y0, x1, y1 int) {
	n := e.lineNum()
	x, y, w, h := e.Rects.Rect(i / n)
	part := float64(i % n)
	theta0 := 2 * math.Pi * part / float64(n)
	theta1 := 2 * math.Pi * (part + 1) / float64(n)
	if theta0 < e.angle0 || e.angle1 < theta1 {
		return 0, 0, 0, 0
	}
	theta0 = math.Max(theta0, e.angle0)
	theta1 = math.Min(theta1, e.angle1)
	theta0 = math.Mod(theta0, 2*math.Pi)
	theta1 = math.Mod(theta1, 2*math.Pi)
	fy0, fx0 := math.Sincos(theta0)
	fy1, fx1 := math.Sincos(theta1)
	hw, hh := (float64(w)-1)/2, (float64(h)-1)/2
	fx0 = fx0*hw + hw + float64(x)
	fx1 = fx1*hw + hw + float64(x)
	fy0 = fy0*hh + hh + float64(y)
	fy1 = fy1*hh + hh + float64(y)
	// TODO: The last fy1 may differ from first fy0 with very slightly difference,
	// which makes the lack of 1 pixel.
	return round(fx0), round(fy0), round(fx1), round(fy1)
}

func (e *ellipsesLines) Color(i int) color.Color {
	return e.Rects.Color(i / e.lineNum())
}

// TODO: This is same as ebiten.rect.
type rect struct {
	x, y          int
	width, height int
	color         color.Color
}

func (r *rect) Len() int {
	return 1
}

func (r *rect) Rect(i int) (x, y, width, height int) {
	return r.x, r.y, r.width, r.height
}

func (r *rect) Color(i int) color.Color {
	return r.color
}

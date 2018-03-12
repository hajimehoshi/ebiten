// Copyright 2014 Hajime Hoshi
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

package affine

import (
	"fmt"
	"math"
)

// GeoMDim is a dimension of a GeoM.
const GeoMDim = 3

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The nil and initial value is identity.
type GeoM struct {
	a_1 float64 // The actual 'a' value minus 1
	b   float64
	c   float64
	d_1 float64 // The actual 'd' value minus 1
	tx  float64
	ty  float64
}

func (g *GeoM) Apply(x, y float64) (x2, y2 float64) {
	if g == nil {
		return x, y
	}
	return (g.a_1+1)*x + g.b*y + g.tx, g.c*x + (g.d_1+1)*y + g.ty
}

func (g *GeoM) Apply32(x, y float64) (x2, y2 float32) {
	if g == nil {
		return float32(x), float32(y)
	}
	return float32((g.a_1+1)*x + g.b*y + g.tx), float32(g.c*x + (g.d_1+1)*y + g.ty)
}

func (g *GeoM) Elements() (a, b, c, d, tx, ty float64) {
	if g == nil {
		return 1, 0, 0, 1, 0, 0
	}
	return g.a_1 + 1, g.b, g.c, g.d_1 + 1, g.tx, g.ty
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) *GeoM {
	a, b, c, d, tx, ty := 1.0, 0.0, 0.0, 1.0, 0.0, 0.0
	if g != nil {
		a, b, c, d, tx, ty = g.a_1+1, g.b, g.c, g.d_1+1, g.tx, g.ty
	}
	switch {
	case i == 0 && j == 0:
		a = element
	case i == 0 && j == 1:
		b = element
	case i == 0 && j == 2:
		tx = element
	case i == 1 && j == 0:
		c = element
	case i == 1 && j == 1:
		d = element
	case i == 1 && j == 2:
		ty = element
	default:
		panic(fmt.Sprintf("affine: i or j is out of index: (%d, %d)", i, j))
	}
	return &GeoM{
		a_1: a - 1,
		b:   b,
		c:   c,
		d_1: d - 1,
		tx:  tx,
		ty:  ty,
	}
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other *GeoM) *GeoM {
	if g == nil {
		return other
	}
	if other == nil {
		return g
	}

	return &GeoM{
		a_1: (other.a_1+1)*(g.a_1+1) + other.b*g.c - 1,
		b:   (other.a_1+1)*g.b + other.b*(g.d_1+1),
		tx:  (other.a_1+1)*g.tx + other.b*g.ty + other.tx,
		c:   other.c*(g.a_1+1) + (other.d_1+1)*g.c,
		d_1: other.c*g.b + (other.d_1+1)*(g.d_1+1) - 1,
		ty:  other.c*g.tx + (other.d_1+1)*g.ty + other.ty,
	}
}

// Add is deprecated.
func (g *GeoM) Add(other *GeoM) *GeoM {
	if g == nil {
		g = &GeoM{}
	}
	if other == nil {
		other = &GeoM{}
	}
	return &GeoM{
		a_1: (g.a_1 + 1) + (other.a_1 + 1) - 1,
		b:   g.b + other.b,
		c:   g.c + other.c,
		d_1: (g.d_1 + 1) + (other.d_1 + 1) - 1,
		tx:  g.tx + other.tx,
		ty:  g.ty + other.ty,
	}
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) *GeoM {
	if g == nil {
		return &GeoM{
			a_1: x - 1,
			b:   0,
			c:   0,
			d_1: y - 1,
		}
	}
	return &GeoM{
		a_1: (g.a_1+1)*x - 1,
		b:   g.b * x,
		tx:  g.tx * x,
		c:   g.c * y,
		d_1: (g.d_1+1)*y - 1,
		ty:  g.ty * y,
	}
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) *GeoM {
	if g == nil {
		return &GeoM{
			tx: tx,
			ty: ty,
		}
	}
	return &GeoM{
		a_1: g.a_1,
		b:   g.b,
		c:   g.c,
		d_1: g.d_1,
		tx:  g.tx + tx,
		ty:  g.ty + ty,
	}
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) *GeoM {
	sin, cos := math.Sincos(theta)
	if g == nil {
		return &GeoM{
			a_1: cos - 1,
			b:   -sin,
			c:   sin,
			d_1: cos - 1,
		}
	}
	return &GeoM{
		a_1: cos*(g.a_1+1) - sin*g.c - 1,
		b:   cos*g.b - sin*(g.d_1+1),
		tx:  cos*g.tx - sin*g.ty,
		c:   sin*(g.a_1+1) + cos*g.c,
		d_1: sin*g.b + cos*(g.d_1+1) - 1,
		ty:  sin*g.tx + cos*g.ty,
	}
}

func (g *GeoM) det() float64 {
	if g == nil {
		return 1
	}
	return (g.a_1+1)*(g.d_1+1) - g.b*g.c
}

func (g *GeoM) IsInvertible() bool {
	return g.det() != 0
}

func (g *GeoM) Invert() *GeoM {
	if g == nil {
		return nil
	}
	det := g.det()
	if det == 0 {
		panic("affine: g is not invertible")
	}
	return &GeoM{
		a_1: ((g.d_1 + 1) / det) - 1,
		b:   -g.b / det,
		c:   -g.c / det,
		d_1: ((g.a_1 + 1) / det) - 1,
		tx:  (-(g.d_1+1)*g.tx + g.b*g.ty) / det,
		ty:  (g.c*g.tx + -(g.a_1+1)*g.ty) / det,
	}
}

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
// The initial value is identity.
type GeoM struct {
	a  float64
	b  float64
	c  float64
	d  float64
	tx float64
	ty float64
}

func (g *GeoM) Apply(x, y float64) (x2, y2 float64) {
	if g == nil {
		return x, y
	}
	return g.a*x + g.b*y + g.tx, g.c*x + g.d*y + g.ty
}

func (g *GeoM) Apply32(x, y float64) (x2, y2 float32) {
	if g == nil {
		return float32(x), float32(y)
	}
	return float32(g.a*x + g.b*y + g.tx), float32(g.c*x + g.d*y + g.ty)
}

func (g *GeoM) Elements() (a, b, c, d, tx, ty float64) {
	if g == nil {
		return 1, 0, 0, 1, 0, 0
	}
	return g.a, g.b, g.c, g.d, g.tx, g.ty
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) *GeoM {
	a, b, c, d, tx, ty := 1.0, 0.0, 0.0, 1.0, 0.0, 0.0
	if g != nil {
		a, b, c, d, tx, ty = g.a, g.b, g.c, g.d, g.tx, g.ty
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
		a:  a,
		b:  b,
		c:  c,
		d:  d,
		tx: tx,
		ty: ty,
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
		a:  other.a*g.a + other.b*g.c,
		b:  other.a*g.b + other.b*g.d,
		tx: other.a*g.tx + other.b*g.ty + other.tx,
		c:  other.c*g.a + other.d*g.c,
		d:  other.c*g.b + other.d*g.d,
		ty: other.c*g.tx + other.d*g.ty + other.ty,
	}
}

// Add is deprecated.
func (g *GeoM) Add(other *GeoM) *GeoM {
	if g == nil {
		g = &GeoM{1, 0, 0, 1, 0, 0}
	}
	if other == nil {
		other = &GeoM{1, 0, 0, 1, 0, 0}
	}
	return &GeoM{
		a:  g.a + other.a,
		b:  g.b + other.b,
		c:  g.c + other.c,
		d:  g.d + other.d,
		tx: g.tx + other.tx,
		ty: g.ty + other.ty,
	}
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) *GeoM {
	if g == nil {
		return &GeoM{
			a:  x,
			b:  0,
			c:  0,
			d:  y,
			tx: 0,
			ty: 0,
		}
	}
	return &GeoM{
		a:  g.a * x,
		b:  g.b * x,
		tx: g.tx * x,
		c:  g.c * y,
		d:  g.d * y,
		ty: g.ty * y,
	}
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) *GeoM {
	if g == nil {
		return &GeoM{
			a:  1,
			b:  0,
			c:  0,
			d:  1,
			tx: tx,
			ty: ty,
		}
	}
	return &GeoM{
		a:  g.a,
		b:  g.b,
		c:  g.c,
		d:  g.d,
		tx: g.tx + tx,
		ty: g.ty + ty,
	}
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) *GeoM {
	sin, cos := math.Sincos(theta)
	if g == nil {
		return &GeoM{
			a:  cos,
			b:  -sin,
			c:  sin,
			d:  cos,
			tx: 0,
			ty: 0,
		}
	}
	return &GeoM{
		a:  cos*g.a - sin*g.c,
		b:  cos*g.b - sin*g.d,
		tx: cos*g.tx - sin*g.ty,
		c:  sin*g.a + cos*g.c,
		d:  sin*g.b + cos*g.d,
		ty: sin*g.tx + cos*g.ty,
	}
}

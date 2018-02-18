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

type geoMImpl struct {
	a  float64
	b  float64
	c  float64
	d  float64
	tx float64
	ty float64
}

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	impl *geoMImpl
}

func (g *GeoM) Reset() {
	g.impl = nil
}

func (g *GeoM) Apply(x, y float64) (x2, y2 float64) {
	if g.impl == nil {
		return x, y
	}
	i := g.impl
	return i.a*x + i.b*y + i.tx, i.c*x + i.d*y + i.ty
}

func (g *GeoM) Apply32(x, y float64) (x2, y2 float32) {
	if g.impl == nil {
		return float32(x), float32(y)
	}
	i := g.impl
	return float32(i.a*x + i.b*y + i.tx), float32(i.c*x + i.d*y + i.ty)
}

func (g *GeoM) Elements() (a, b, c, d, tx, ty float64) {
	if g.impl == nil {
		return 1, 0, 0, 1, 0, 0
	}
	i := g.impl
	return i.a, i.b, i.c, i.d, i.tx, i.ty
}

func (g *GeoM) init() {
	g.impl = &geoMImpl{
		a:  1,
		b:  0,
		c:  0,
		d:  1,
		tx: 0,
		ty: 0,
	}
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	if g.impl == nil {
		g.init()
	}
	a, b, c, d, tx, ty := g.impl.a, g.impl.b, g.impl.c, g.impl.d, g.impl.tx, g.impl.ty
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
	g.impl = &geoMImpl{
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
func (g *GeoM) Concat(other *GeoM) {
	if g.impl == nil {
		g.init()
	}
	if other.impl == nil {
		other.init()
	}

	i := g.impl
	oi := other.impl
	g.impl = &geoMImpl{
		a:  oi.a*i.a + oi.b*i.c,
		b:  oi.a*i.b + oi.b*i.d,
		tx: oi.a*i.tx + oi.b*i.ty + oi.tx,
		c:  oi.c*i.a + oi.d*i.c,
		d:  oi.c*i.b + oi.d*i.d,
		ty: oi.c*i.tx + oi.d*i.ty + oi.ty,
	}
}

// Add is deprecated.
func (g *GeoM) Add(other GeoM) {
	if g.impl == nil {
		g.init()
	}
	if other.impl == nil {
		other.init()
	}
	g.impl = &geoMImpl{
		a:  g.impl.a + other.impl.a,
		b:  g.impl.b + other.impl.b,
		c:  g.impl.c + other.impl.c,
		d:  g.impl.d + other.impl.d,
		tx: g.impl.tx + other.impl.tx,
		ty: g.impl.ty + other.impl.ty,
	}
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	if g.impl == nil {
		g.impl = &geoMImpl{
			a:  x,
			b:  0,
			c:  0,
			d:  y,
			tx: 0,
			ty: 0,
		}
		return
	}
	g.impl = &geoMImpl{
		a:  g.impl.a * x,
		b:  g.impl.b * x,
		tx: g.impl.tx * x,
		c:  g.impl.c * y,
		d:  g.impl.d * y,
		ty: g.impl.ty * y,
	}
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	if g.impl == nil {
		g.impl = &geoMImpl{
			a:  1,
			b:  0,
			c:  0,
			d:  1,
			tx: tx,
			ty: ty,
		}
		return
	}
	g.impl = &geoMImpl{
		a:  g.impl.a,
		b:  g.impl.b,
		c:  g.impl.c,
		d:  g.impl.d,
		tx: g.impl.tx + tx,
		ty: g.impl.ty + ty,
	}
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	if g.impl == nil {
		g.impl = &geoMImpl{
			a:  cos,
			b:  -sin,
			c:  sin,
			d:  cos,
			tx: 0,
			ty: 0,
		}
		return
	}
	g.impl = &geoMImpl{
		a:  cos*g.impl.a - sin*g.impl.c,
		b:  cos*g.impl.b - sin*g.impl.d,
		tx: cos*g.impl.tx - sin*g.impl.ty,
		c:  sin*g.impl.a + cos*g.impl.c,
		d:  sin*g.impl.b + cos*g.impl.d,
		ty: sin*g.impl.tx + cos*g.impl.ty,
	}
}

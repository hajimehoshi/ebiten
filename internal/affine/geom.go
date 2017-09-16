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
	"math"
)

// GeoMDim is a dimension of a GeoM.
const GeoMDim = 3

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	a      float64
	b      float64
	c      float64
	d      float64
	tx     float64
	ty     float64
	inited bool
}

func (g *GeoM) Reset() {
	g.inited = false
}

func (g *GeoM) Elements() (a, b, c, d, tx, ty float64) {
	if !g.inited {
		return 1, 0, 0, 1, 0, 0
	}
	return g.a, g.b, g.c, g.d, g.tx, g.ty
}

func (g *GeoM) init() {
	g.a = 1
	g.b = 0
	g.c = 0
	g.d = 1
	g.tx = 0
	g.ty = 0
	g.inited = true
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	if !g.inited {
		g.init()
	}
	switch {
	case i == 0 && j == 0:
		g.a = element
	case i == 0 && j == 1:
		g.b = element
	case i == 0 && j == 2:
		g.tx = element
	case i == 1 && j == 0:
		g.c = element
	case i == 1 && j == 1:
		g.d = element
	case i == 1 && j == 2:
		g.ty = element
	default:
		panic("affine: i or j is out of index")
	}
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other *GeoM) {
	if !g.inited {
		g.init()
	}
	if !other.inited {
		other.init()
	}
	a, b, c, d, tx, ty := g.a, g.b, g.c, g.d, g.tx, g.ty
	g.a = other.a*a + other.b*c
	g.b = other.a*b + other.b*d
	g.tx = other.a*tx + other.b*ty + other.tx
	g.c = other.c*a + other.d*c
	g.d = other.c*b + other.d*d
	g.ty = other.c*tx + other.d*ty + other.ty
}

// Add is deprecated.
func (g *GeoM) Add(other GeoM) {
	if !g.inited {
		g.init()
	}
	if !other.inited {
		other.init()
	}
	g.a += other.a
	g.b += other.b
	g.c += other.c
	g.d += other.d
	g.tx += other.tx
	g.ty += other.ty
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	if !g.inited {
		g.a = x
		g.b = 0
		g.c = 0
		g.d = y
		g.tx = 0
		g.ty = 0
		g.inited = true
		return
	}
	g.a *= x
	g.b *= x
	g.tx *= x
	g.c *= y
	g.d *= y
	g.ty *= y
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	if !g.inited {
		g.a = 1
		g.b = 0
		g.c = 0
		g.d = 1
		g.tx = tx
		g.ty = ty
		g.inited = true
		return
	}
	g.tx += tx
	g.ty += ty
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	if !g.inited {
		g.a = cos
		g.b = -sin
		g.c = sin
		g.d = cos
		g.tx = 0
		g.ty = 0
		g.inited = true
		return
	}
	a, b, c, d, tx, ty := g.a, g.b, g.c, g.d, g.tx, g.ty
	g.a = cos*a - sin*c
	g.b = cos*b - sin*d
	g.tx = cos*tx - sin*ty
	g.c = sin*a + cos*c
	g.d = sin*b + cos*d
	g.ty = sin*tx + cos*ty
}

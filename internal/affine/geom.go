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

var (
	geoMIdentityElements = []float64{
		1, 0, 0,
		0, 1, 0,
	}
)

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	// When elements is empty, this matrix is identity.
	// elements is immutable and a new array must be created when updating.
	elements []float64
}

func (g *GeoM) UnsafeElements() []float64 {
	if g.elements == nil {
		g.elements = geoMIdentityElements
	}
	return g.elements
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	if g.elements == nil {
		g.elements = geoMIdentityElements
	}
	es := make([]float64, len(g.elements))
	copy(es, g.elements)
	es[i*GeoMDim+j] = element
	g.elements = es
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other GeoM) {
	if g.elements == nil {
		g.elements = geoMIdentityElements
	}
	if other.elements == nil {
		other.elements = geoMIdentityElements
	}
	g.elements = mul(other.elements, g.elements, GeoMDim)
}

// Add is deprecated.
func (g *GeoM) Add(other GeoM) {
	if g.elements == nil {
		g.elements = geoMIdentityElements
	}
	if other.elements == nil {
		other.elements = geoMIdentityElements
	}
	g.elements = add(other.elements, g.elements, GeoMDim)
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	if g.elements == nil {
		g.elements = []float64{
			x, 0, 0,
			0, y, 0,
		}
		return
	}
	es := make([]float64, len(g.elements))
	copy(es, g.elements)
	for i := 0; i < GeoMDim; i++ {
		es[i] *= x
		es[i+GeoMDim] *= y
	}
	g.elements = es
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	if g.elements == nil {
		g.elements = []float64{
			1, 0, tx,
			0, 1, ty,
		}
		return
	}
	es := make([]float64, len(g.elements))
	copy(es, g.elements)
	es[2] += tx
	es[2+GeoMDim] += ty
	g.elements = es
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	g.Concat(GeoM{
		elements: []float64{
			cos, -sin, 0,
			sin, cos, 0,
		},
	})
}

// ScaleGeo is deprecated as of 1.2.0-alpha. Use Scale instead.
func ScaleGeo(x, y float64) GeoM {
	g := GeoM{}
	g.Scale(x, y)
	return g
}

// TranslateGeo is deprecated as of 1.2.0-alpha. Use Translate instead.
func TranslateGeo(tx, ty float64) GeoM {
	g := GeoM{}
	g.Translate(tx, ty)
	return g
}

// RotateGeo is deprecated as of 1.2.0-alpha. Use Rotate instead.
func RotateGeo(theta float64) GeoM {
	g := GeoM{}
	g.Rotate(theta)
	return g
}

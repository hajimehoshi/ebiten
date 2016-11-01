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
	initialized bool
	es          [(GeoMDim - 1) * GeoMDim]float64
}

func (g *GeoM) dim() int {
	return GeoMDim
}

func (g *GeoM) initialize() {
	g.initialized = true
	for i := 0; i < GeoMDim-1; i++ {
		g.es[i*GeoMDim+i] = 1
	}
}

// Element returns a value of a matrix at (i, j).
func (g *GeoM) Element(i, j int) float64 {
	if !g.initialized {
		if i == j {
			return 1
		}
		return 0
	}
	return g.es[i*GeoMDim+j]
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other GeoM) {
	if !g.initialized {
		g.initialize()
	}
	result := GeoM{}
	mul(&other, g, &result)
	*g = result
}

// Add adds a geometry matrix with the other geometry matrix.
func (g *GeoM) Add(other GeoM) {
	if !g.initialized {
		g.initialize()
	}
	result := GeoM{}
	add(&other, g, &result)
	*g = result
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	if !g.initialized {
		g.initialize()
	}
	for i := 0; i < GeoMDim; i++ {
		g.es[i] *= x
		g.es[GeoMDim+i] *= y
	}
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	if !g.initialized {
		g.initialize()
	}
	g.es[2] += tx
	g.es[GeoMDim+2] += ty
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	g.Concat(GeoM{
		initialized: true,
		es: [...]float64{
			cos, -sin, 0,
			sin, cos, 0,
		},
	})
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	if !g.initialized {
		g.initialize()
	}
	g.es[i*GeoMDim+j] = element
}

// ScaleGeo is deprecated as of 1.2.0-alpha. Use Scale instead.
func ScaleGeo(x, y float64) GeoM {
	return GeoM{
		initialized: true,
		es: [...]float64{
			x, 0, 0,
			0, y, 0,
		},
	}
}

// TranslateGeo is deprecated as of 1.2.0-alpha. Use Translate instead.
func TranslateGeo(tx, ty float64) GeoM {
	return GeoM{
		initialized: true,
		es: [...]float64{
			1, 0, tx,
			0, 1, ty,
		},
	}
}

// RotateGeo is deprecated as of 1.2.0-alpha. Use Rotate instead.
func RotateGeo(theta float64) GeoM {
	sin, cos := math.Sincos(theta)
	return GeoM{
		initialized: true,
		es: [...]float64{
			cos, -sin, 0,
			sin, cos, 0,
		},
	}
}

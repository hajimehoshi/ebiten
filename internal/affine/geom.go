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

func geoMValueString(values [GeoMDim - 1][GeoMDim]float64) string {
	b := make([]uint8, 0, (GeoMDim-1)*(GeoMDim)*8)
	for i := 0; i < GeoMDim-1; i++ {
		for j := 0; j < GeoMDim; j++ {
			b = append(b, uint64ToBytes(math.Float64bits(values[i][j]))...)
		}
	}
	return string(b)
}

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	// When values is empty, this matrix is identity.
	values string
}

func (g *GeoM) dim() int {
	return GeoMDim
}

func (g *GeoM) Elements() []float64 {
	return elements(g.values, GeoMDim)
}

func (g *GeoM) element(i, j int) float64 {
	return g.Elements()[i*GeoMDim+j]
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	g.values = setElement(g.values, GeoMDim, i, j, element)
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other GeoM) {
	result := GeoM{}
	mul(&other, g, &result)
	*g = result
}

// Add is deprecated.
func (g *GeoM) Add(other GeoM) {
	result := GeoM{}
	add(&other, g, &result)
	*g = result
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	v := elements(g.values, GeoMDim)
	for i := 0; i < GeoMDim; i++ {
		v[i] *= x
		v[i+GeoMDim] *= y
	}
	g.values = setElements(v, GeoMDim)
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	v := elements(g.values, GeoMDim)
	v[2] += tx
	v[2+GeoMDim] += ty
	g.values = setElements(v, GeoMDim)
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	g.Concat(GeoM{
		values: geoMValueString([GeoMDim - 1][GeoMDim]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		}),
	})
}

// ScaleGeo is deprecated as of 1.2.0-alpha. Use Scale instead.
func ScaleGeo(x, y float64) GeoM {
	return GeoM{
		values: geoMValueString([GeoMDim - 1][GeoMDim]float64{
			{x, 0, 0},
			{0, y, 0},
		}),
	}
}

// TranslateGeo is deprecated as of 1.2.0-alpha. Use Translate instead.
func TranslateGeo(tx, ty float64) GeoM {
	return GeoM{
		values: geoMValueString([GeoMDim - 1][GeoMDim]float64{
			{1, 0, tx},
			{0, 1, ty},
		}),
	}
}

// RotateGeo is deprecated as of 1.2.0-alpha. Use Rotate instead.
func RotateGeo(theta float64) GeoM {
	sin, cos := math.Sincos(theta)
	return GeoM{
		values: geoMValueString([GeoMDim - 1][GeoMDim]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		}),
	}
}

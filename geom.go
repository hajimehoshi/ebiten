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

package ebiten

import (
	"math"
)

// GeoMDim is a dimension of a GeoM.
const GeoMDim = 3

var geoMI = GeoM{
	initialized: true,
	es: [2][3]float64{
		{1, 0, 0},
		{0, 1, 0},
	},
}

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	initialized bool
	es          [GeoMDim - 1][GeoMDim]float64
}

func (g GeoM) dim() int {
	return GeoMDim
}

// Element returns a value of a matrix at (i, j).
func (g GeoM) Element(i, j int) float64 {
	if !g.initialized {
		if i == j {
			return 1
		}
		return 0
	}
	return g.es[i][j]
}

// Concat multiplies a geometry matrix with the other geometry matrix.
func (g *GeoM) Concat(other GeoM) {
	if !g.initialized {
		*g = geoMI
	}
	result := GeoM{}
	mul(&other, g, &result)
	*g = result
}

// Add adds a geometry matrix with the other geometry matrix.
func (g *GeoM) Add(other GeoM) {
	if !g.initialized {
		*g = geoMI
	}
	result := GeoM{}
	add(&other, g, &result)
	*g = result
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	if !g.initialized {
		*g = geoMI
	}
	g.es[i][j] = element
}

// ScaleGeo returns a matrix that scales a geometry matrix by (x, y).
func ScaleGeo(x, y float64) GeoM {
	return GeoM{
		initialized: true,
		es: [2][3]float64{
			{x, 0, 0},
			{0, y, 0},
		},
	}
}

// TranslateGeo returns a matrix taht translates a geometry matrix by (tx, ty).
func TranslateGeo(tx, ty float64) GeoM {
	return GeoM{
		initialized: true,
		es: [2][3]float64{
			{1, 0, tx},
			{0, 1, ty},
		},
	}
}

// RotateGeo returns a matrix that rotates a geometry matrix by theta.
func RotateGeo(theta float64) GeoM {
	sin, cos := math.Sincos(theta)
	return GeoM{
		initialized: true,
		es: [2][3]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		},
	}
}

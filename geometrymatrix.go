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

// GeometryMatrixDim is a dimension of a GeometryMatrix.
const GeometryMatrixDim = 3

// A GeometryMatrix represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeometryMatrix struct {
	// es represents elements.
	es *[GeometryMatrixDim - 1][GeometryMatrixDim]float64
}

func geometryMatrixEsI() *[GeometryMatrixDim - 1][GeometryMatrixDim]float64 {
	return &[GeometryMatrixDim - 1][GeometryMatrixDim]float64{
		{1, 0, 0},
		{0, 1, 0},
	}
}

func (g GeometryMatrix) dim() int {
	return GeometryMatrixDim
}

// Element returns a value of a matrix at (i, j).
func (g GeometryMatrix) Element(i, j int) float64 {
	if g.es == nil {
		if i == j {
			return 1
		}
		return 0
	}
	return g.es[i][j]
}

// Concat multiplies a geometry matrix with the other geometry matrix.
func (g *GeometryMatrix) Concat(other GeometryMatrix) {
	if g.es == nil {
		g.es = geometryMatrixEsI()
	}
	result := GeometryMatrix{}
	mul(&other, g, &result)
	*g = result
}

// Add adds a geometry matrix with the other geometry matrix.
func (g *GeometryMatrix) Add(other GeometryMatrix) {
	if g.es == nil {
		g.es = geometryMatrixEsI()
	}
	result := GeometryMatrix{}
	add(&other, g, &result)
	*g = result
}

// SetElement sets an element at (i, j).
func (g *GeometryMatrix) SetElement(i, j int, element float64) {
	if g.es == nil {
		g.es = geometryMatrixEsI()
	}
	g.es[i][j] = element
}

// ScaleGeometry returns a matrix that scales a geometry matrix by (x, y).
func ScaleGeometry(x, y float64) GeometryMatrix {
	return GeometryMatrix{
		es: &[2][3]float64{
			{x, 0, 0},
			{0, y, 0},
		},
	}
}

// TranslateGeometry returns a matrix taht translates a geometry matrix by (tx, ty).
func TranslateGeometry(tx, ty float64) GeometryMatrix {
	return GeometryMatrix{
		es: &[2][3]float64{
			{1, 0, tx},
			{0, 1, ty},
		},
	}
}

// RotateGeometry returns a matrix that rotates a geometry matrix by theta.
func RotateGeometry(theta float64) GeometryMatrix {
	sin, cos := math.Sincos(theta)
	return GeometryMatrix{
		es: &[2][3]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		},
	}
}

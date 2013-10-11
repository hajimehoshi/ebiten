// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package matrix

import (
	"math"
)

const geometryDim = 3

type Geometry struct {
	Elements [geometryDim - 1][geometryDim]float64
}

func IdentityGeometry() Geometry {
	return Geometry{
		[geometryDim - 1][geometryDim]float64{
			{1, 0, 0},
			{0, 1, 0},
		},
	}
}

func (matrix *Geometry) Dim() int {
	return geometryDim
}

func (matrix *Geometry) Concat(other Geometry) {
	result := Geometry{}
	mul(&other, matrix, &result)
	*matrix = result
}

func (matrix *Geometry) IsIdentity() bool {
	return isIdentity(matrix)
}

func (matrix *Geometry) element(i, j int) float64 {
	return matrix.Elements[i][j]
}

func (matrix *Geometry) setElement(i, j int, element float64) {
	matrix.Elements[i][j] = element
}

func (matrix *Geometry) Translate(tx, ty float64) {
	matrix.Elements[0][2] += tx
	matrix.Elements[1][2] += ty
}

func (matrix *Geometry) Scale(x, y float64) {
	matrix.Elements[0][0] *= x
	matrix.Elements[0][1] *= x
	matrix.Elements[0][2] *= x
	matrix.Elements[1][0] *= y
	matrix.Elements[1][1] *= y
	matrix.Elements[1][2] *= y
}

func (matrix *Geometry) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	rotate := Geometry{
		[2][3]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		},
	}
	matrix.Concat(rotate)
}

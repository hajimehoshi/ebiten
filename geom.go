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
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
)

// GeoMDim is a dimension of a GeoM.
const GeoMDim = affine.GeoMDim

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	impl *affine.GeoM
}

// String returns a string representation of GeoM.
func (g *GeoM) String() string {
	a, b, c, d, tx, ty := g.impl.Elements()
	return fmt.Sprintf("[[%f, %f, %f], [%f, %f, %f]]", a, b, tx, c, d, ty)
}

// Reset resets the GeoM as identity.
func (g *GeoM) Reset() {
	g.impl = nil
}

// Apply pre-multiplies a vector (x, y, 1) by the matrix.
// In other words, Apply calculates GeoM * (x, y, 1)^T.
// The return value is x and y values of the result vector.
func (g *GeoM) Apply(x, y float64) (x2, y2 float64) {
	return g.impl.Apply(x, y)
}

// Element returns a value of a matrix at (i, j).
func (g *GeoM) Element(i, j int) float64 {
	a, b, c, d, tx, ty := g.impl.Elements()
	switch {
	case i == 0 && j == 0:
		return a
	case i == 0 && j == 1:
		return b
	case i == 0 && j == 2:
		return tx
	case i == 1 && j == 0:
		return c
	case i == 1 && j == 1:
		return d
	case i == 1 && j == 2:
		return ty
	default:
		panic("ebiten: i or j is out of index")
	}
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other GeoM) {
	g.impl = g.impl.Concat(other.impl)
}

// Add is deprecated as of 1.5.0-alpha.
// Note that this doesn't make sense as an operation for affine matrices.
func (g *GeoM) Add(other GeoM) {
	g.impl = g.impl.Add(other.impl)
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	g.impl = g.impl.Scale(x, y)
}

// Translate translates the matrix by (tx, ty).
func (g *GeoM) Translate(tx, ty float64) {
	g.impl = g.impl.Translate(tx, ty)
}

// IsInvertible returns a boolean value indicating
// whether the matrix g is invertible or not.
func (g *GeoM) IsInvertible() bool {
	return g.impl.IsInvertible()
}

// Invert inverts the matrix.
// If g is not invertible, Invert panics.
func (g *GeoM) Invert() {
	g.impl = g.impl.Invert()
}

// Rotate rotates the matrix by theta.
// The unit is radian.
func (g *GeoM) Rotate(theta float64) {
	g.impl = g.impl.Rotate(theta)
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	g.impl = g.impl.SetElement(i, j, element)
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

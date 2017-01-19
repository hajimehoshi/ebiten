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
	"github.com/hajimehoshi/ebiten/internal/affine"
)

// GeoMDim is a dimension of a GeoM.
const GeoMDim = affine.GeoMDim

// A GeoM represents a matrix to transform geometry when rendering an image.
//
// The initial value is identity.
type GeoM struct {
	impl affine.GeoM
}

// Element returns a value of a matrix at (i, j).
func (g *GeoM) Element(i, j int) float64 {
	return g.impl.Elements()[i*affine.GeoMDim+j]
}

// Concat multiplies a geometry matrix with the other geometry matrix.
// This is same as muptiplying the matrix other and the matrix g in this order.
func (g *GeoM) Concat(other GeoM) {
	g.impl.Concat(other.impl)
}

// Add is deprecated as of 1.5.0-alpha.
// Note that this doesn't make sense as an operation for affine matrices.
func (g *GeoM) Add(other GeoM) {
	g.impl.Add(other.impl)
}

// Scale scales the matrix by (x, y).
func (g *GeoM) Scale(x, y float64) {
	g.impl.Scale(x, y)
}

// Translate translates the matrix by (x, y).
func (g *GeoM) Translate(tx, ty float64) {
	g.impl.Translate(tx, ty)
}

// Rotate rotates the matrix by theta.
func (g *GeoM) Rotate(theta float64) {
	g.impl.Rotate(theta)
}

// SetElement sets an element at (i, j).
func (g *GeoM) SetElement(i, j int, element float64) {
	g.impl.SetElement(i, j, element)
}

// ScaleGeo is deprecated as of 1.2.0-alpha. Use Scale instead.
func ScaleGeo(x, y float64) GeoM {
	return GeoM{affine.ScaleGeo(x, y)}
}

// TranslateGeo is deprecated as of 1.2.0-alpha. Use Translate instead.
func TranslateGeo(tx, ty float64) GeoM {
	return GeoM{affine.TranslateGeo(tx, ty)}
}

// RotateGeo is deprecated as of 1.2.0-alpha. Use Rotate instead.
func RotateGeo(theta float64) GeoM {
	return GeoM{affine.RotateGeo(theta)}
}

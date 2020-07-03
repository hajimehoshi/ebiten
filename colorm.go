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

	"image/color"

	"github.com/hajimehoshi/ebiten/internal/affine"
)

// ColorMDim is a dimension of a ColorM.
const ColorMDim = affine.ColorMDim

// A ColorM represents a matrix to transform coloring when rendering an image.
//
// A ColorM is applied to the straight alpha color
// while an Image's pixels' format is alpha premultiplied.
// Before applying a matrix, a color is un-multiplied, and after applying the matrix,
// the color is multiplied again.
//
// The initial value is identity.
type ColorM struct {
	impl *affine.ColorM

	_ [0]func() // Marks as non-comparable.
}

// String returns a string representation of ColorM.
func (c *ColorM) String() string {
	b, t := c.impl.UnsafeElements()
	return fmt.Sprintf("[[%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f]]",
		b[0], b[4], b[8], b[12], t[0],
		b[1], b[5], b[9], b[13], t[1],
		b[2], b[6], b[10], b[14], t[2],
		b[3], b[7], b[11], b[15], t[3])
}

// Reset resets the ColorM as identity.
func (c *ColorM) Reset() {
	c.impl = nil
}

// Apply pre-multiplies a vector (r, g, b, a, 1) by the matrix
// where r, g, b, and a are clr's values in straight-alpha format.
// In other words, Apply calculates ColorM * (r, g, b, a, 1)^T.
func (c *ColorM) Apply(clr color.Color) color.Color {
	return c.impl.Apply(clr)
}

// Concat multiplies a color matrix with the other color matrix.
// This is same as muptiplying the matrix other and the matrix c in this order.
func (c *ColorM) Concat(other ColorM) {
	c.impl = c.impl.Concat(other.impl)
}

// Add adds a matrix, but in a wrong way.
//
// Deprecated: (as of 1.5.0) Do not use this.
//
// Note that this doesn't make sense as an operation for affine matrices.
func (c *ColorM) Add(other ColorM) {
	c.impl = c.impl.Add(other.impl)
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float64) {
	c.impl = c.impl.Scale(float32(r), float32(g), float32(b), float32(a))
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float64) {
	c.impl = c.impl.Translate(float32(r), float32(g), float32(b), float32(a))
}

// RotateHue rotates the hue.
// theta represents rotating angle in radian.
func (c *ColorM) RotateHue(theta float64) {
	c.ChangeHSV(theta, 1, 1)
}

// ChangeHSV changes HSV (Hue-Saturation-Value) values.
// hueTheta is a radian value to rotate hue.
// saturationScale is a value to scale saturation.
// valueScale is a value to scale value (a.k.a. brightness).
//
// This conversion uses RGB to/from YCrCb conversion.
func (c *ColorM) ChangeHSV(hueTheta float64, saturationScale float64, valueScale float64) {
	c.impl = c.impl.ChangeHSV(hueTheta, float32(saturationScale), float32(valueScale))
}

// Element returns a value of a matrix at (i, j).
func (c *ColorM) Element(i, j int) float64 {
	return float64(c.impl.Element(i, j))
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float64) {
	c.impl = c.impl.SetElement(i, j, float32(element))
}

// IsInvertible returns a boolean value indicating
// whether the matrix c is invertible or not.
func (c *ColorM) IsInvertible() bool {
	return c.impl.IsInvertible()
}

// Invert inverts the matrix.
// If c is not invertible, Invert panics.
func (c *ColorM) Invert() {
	c.impl = c.impl.Invert()
}

// Monochrome returns a color matrix for monochrome.
//
// Deprecated: (as of 1.6.0) Use ChangeHSV(0, 0, 1) instead.
func Monochrome() ColorM {
	c := ColorM{}
	c.ChangeHSV(0, 0, 1)
	return c
}

// ScaleColor returns a color matrix for scaling.
//
// Deprecated: (as of 1.2.0) Use Scale instead.
func ScaleColor(r, g, b, a float64) ColorM {
	c := ColorM{}
	c.Scale(r, g, b, a)
	return c
}

// TranslateColor returns a color matrix for translating.
//
// Deprecated: (as of 1.2.0) Use Translate instead.
func TranslateColor(r, g, b, a float64) ColorM {
	c := ColorM{}
	c.Translate(r, g, b, a)
	return c
}

// RotateHue returns a color matrix for chanting the hue.
//
// Deprecated: (as of 1.2.0-alpha) Use RotateHue member function instead.
func RotateHue(theta float64) ColorM {
	c := ColorM{}
	c.RotateHue(theta)
	return c
}

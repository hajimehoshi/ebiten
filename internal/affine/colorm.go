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
	"image/color"
	"math"
)

// ColorMDim is a dimension of a ColorM.
const ColorMDim = 5

var (
	colorMIdentityElements = []float64{
		1, 0, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
	}
)

// A ColorM represents a matrix to transform coloring when rendering an image.
//
// A ColorM is applied to the source alpha color
// while an Image's pixels' format is alpha premultiplied.
// Before applying a matrix, a color is un-multiplied, and after applying the matrix,
// the color is multiplied again.
//
// The initial value is identity.
type ColorM struct {
	// When elements is nil, this matrix is identity.
	// elements is immutable and a new array must be created when updating.
	elements []float64
}

func (c *ColorM) Reset() {
	c.elements = nil
}

func clamp(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < 0 {
		return 0
	}
	return x
}

func (c *ColorM) Apply(clr color.Color) color.Color {
	if c.elements == nil {
		return clr
	}
	r, g, b, a := clr.RGBA()
	rf, gf, bf, af := 0.0, 0.0, 0.0, 0.0
	// Unmultiply alpha
	if a > 0 {
		rf = float64(r) / float64(a)
		gf = float64(g) / float64(a)
		bf = float64(b) / float64(a)
		af = float64(a) / 0xffff
	}
	e := c.elements
	rf2 := e[0]*rf + e[1]*gf + e[2]*bf + e[3]*af + e[4]
	gf2 := e[5]*rf + e[6]*gf + e[7]*bf + e[8]*af + e[9]
	bf2 := e[10]*rf + e[11]*gf + e[12]*bf + e[13]*af + e[14]
	af2 := e[15]*rf + e[16]*gf + e[17]*bf + e[18]*af + e[19]
	rf2 = clamp(rf2)
	gf2 = clamp(gf2)
	bf2 = clamp(bf2)
	af2 = clamp(af2)
	return color.NRGBA64{
		R: uint16(rf2 * 0xffff),
		G: uint16(gf2 * 0xffff),
		B: uint16(bf2 * 0xffff),
		A: uint16(af2 * 0xffff),
	}
}

func (c *ColorM) UnsafeElements() []float64 {
	if c.elements == nil {
		c.elements = colorMIdentityElements
	}
	return c.elements
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float64) {
	if c.elements == nil {
		c.elements = colorMIdentityElements
	}
	es := make([]float64, len(c.elements))
	copy(es, c.elements)
	es[i*ColorMDim+j] = element
	c.elements = es
}

func (c *ColorM) Equals(other *ColorM) bool {
	if c.elements == nil {
		if other.elements == nil {
			return true
		}
		c.elements = colorMIdentityElements
	}
	if other.elements == nil {
		other.elements = colorMIdentityElements
	}
	for i := range c.elements {
		if c.elements[i] != other.elements[i] {
			return false
		}
	}
	return true
}

// Concat multiplies a color matrix with the other color matrix.
// This is same as muptiplying the matrix other and the matrix c in this order.
func (c *ColorM) Concat(other *ColorM) {
	if c.elements == nil {
		c.elements = colorMIdentityElements
	}
	if other.elements == nil {
		other.elements = colorMIdentityElements
	}
	c.elements = mul(other.elements, c.elements, ColorMDim)
}

// Add is deprecated.
func (c *ColorM) Add(other ColorM) {
	if c.elements == nil {
		c.elements = colorMIdentityElements
	}
	if other.elements == nil {
		other.elements = colorMIdentityElements
	}
	c.elements = add(other.elements, c.elements, ColorMDim)
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float64) {
	if c.elements == nil {
		c.elements = []float64{
			r, 0, 0, 0, 0,
			0, g, 0, 0, 0,
			0, 0, b, 0, 0,
			0, 0, 0, a, 0,
		}
		return
	}
	es := make([]float64, len(c.elements))
	copy(es, c.elements)
	for i := 0; i < ColorMDim; i++ {
		es[i] *= r
		es[i+ColorMDim] *= g
		es[i+ColorMDim*2] *= b
		es[i+ColorMDim*3] *= a
	}
	c.elements = es
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float64) {
	if c.elements == nil {
		c.elements = []float64{
			1, 0, 0, 0, r,
			0, 1, 0, 0, g,
			0, 0, 1, 0, b,
			0, 0, 0, 1, a,
		}
		return
	}
	es := make([]float64, len(c.elements))
	copy(es, c.elements)
	es[4] += r
	es[4+ColorMDim] += g
	es[4+ColorMDim*2] += b
	es[4+ColorMDim*3] += a
	c.elements = es
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = ColorM{
		elements: []float64{
			0.2990, 0.5870, 0.1140, 0, 0,
			-0.1687, -0.3313, 0.5000, 0, 0,
			0.5000, -0.4187, -0.0813, 0, 0,
			0, 0, 0, 1, 0,
		},
	}
	yCbCrToRgb = ColorM{
		elements: []float64{
			1, 0, 1.40200, 0, 0,
			1, -0.34414, -0.71414, 0, 0,
			1, 1.77200, 0, 0, 0,
			0, 0, 0, 1, 0,
		},
	}
)

// ChangeHSV changes HSV (Hue-Saturation-Value) elements.
// hueTheta is a radian value to ratate hue.
// saturationScale is a value to scale saturation.
// valueScale is a value to scale value (a.k.a. brightness).
//
// This conversion uses RGB to/from YCrCb conversion.
func (c *ColorM) ChangeHSV(hueTheta float64, saturationScale float64, valueScale float64) {
	sin, cos := math.Sincos(hueTheta)
	c.Concat(&rgbToYCbCr)
	c.Concat(&ColorM{
		elements: []float64{
			1, 0, 0, 0, 0,
			0, cos, -sin, 0, 0,
			0, sin, cos, 0, 0,
			0, 0, 0, 1, 0,
		},
	})
	s := saturationScale
	v := valueScale
	c.Scale(v, s*v, s*v, 1)
	c.Concat(&yCbCrToRgb)
}

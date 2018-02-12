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
	colorMIdentityBody = []float64{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	colorMIdentityTranslate = []float64{
		0, 0, 0, 0,
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
	body      []float64 // TODO: Transpose this to pass this OpenGL easily
	translate []float64
}

func (c *ColorM) Reset() {
	c.body = nil
	c.translate = nil
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
	if c.body == nil {
		return clr
	}
	r, g, b, a := clr.RGBA()
	if a == 0 {
		return color.Transparent
	}
	// Unmultiply alpha
	rf := float64(r) / float64(a)
	gf := float64(g) / float64(a)
	bf := float64(b) / float64(a)
	af := float64(a) / 0xffff
	eb := c.body
	et := c.translate
	rf2 := eb[0]*rf + eb[1]*gf + eb[2]*bf + eb[3]*af + et[0]
	gf2 := eb[4]*rf + eb[5]*gf + eb[6]*bf + eb[7]*af + et[1]
	bf2 := eb[8]*rf + eb[9]*gf + eb[10]*bf + eb[11]*af + et[2]
	af2 := eb[12]*rf + eb[13]*gf + eb[14]*bf + eb[15]*af + et[3]
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

func (c *ColorM) UnsafeElements() ([]float64, []float64) {
	if c.body == nil {
		c.body = colorMIdentityBody
		c.translate = colorMIdentityTranslate
	}
	return c.body, c.translate
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float64) {
	if c.body == nil {
		c.body = colorMIdentityBody
		c.translate = colorMIdentityTranslate
	}
	if j < (ColorMDim - 1) {
		es := make([]float64, len(c.body))
		copy(es, c.body)
		es[i*(ColorMDim-1)+j] = element
		c.body = es
	} else {
		es := make([]float64, len(c.translate))
		copy(es, c.translate)
		es[i] = element
		c.translate = es
	}
}

func (c *ColorM) Equals(other *ColorM) bool {
	if c.body == nil {
		if other.body == nil {
			return true
		}
		c.body = colorMIdentityBody
		c.translate = colorMIdentityTranslate
	}
	if other.body == nil {
		other.body = colorMIdentityBody
		other.translate = colorMIdentityTranslate
	}
	for i := range c.body {
		if c.body[i] != other.body[i] {
			return false
		}
	}
	for i := range c.translate {
		if c.translate[i] != other.translate[i] {
			return false
		}
	}
	return true
}

// Concat multiplies a color matrix with the other color matrix.
// This is same as muptiplying the matrix other and the matrix c in this order.
func (c *ColorM) Concat(other *ColorM) {
	if c.body == nil {
		c.body = colorMIdentityBody
		c.translate = colorMIdentityTranslate
	}
	if other.body == nil {
		other.body = colorMIdentityBody
		other.translate = colorMIdentityTranslate
	}
	c.body = mulSquare(other.body, c.body, ColorMDim-1)

	lhsb := other.body
	lhst := other.translate
	rhst := c.translate
	c.translate = []float64{
		lhsb[0]*rhst[0] + lhsb[1]*rhst[1] + lhsb[2]*rhst[2] + lhsb[3]*rhst[3] + lhst[0],
		lhsb[4]*rhst[0] + lhsb[5]*rhst[1] + lhsb[6]*rhst[2] + lhsb[7]*rhst[3] + lhst[1],
		lhsb[8]*rhst[0] + lhsb[9]*rhst[1] + lhsb[10]*rhst[2] + lhsb[11]*rhst[3] + lhst[2],
		lhsb[12]*rhst[0] + lhsb[13]*rhst[1] + lhsb[14]*rhst[2] + lhsb[15]*rhst[3] + lhst[3],
	}
}

// Add is deprecated.
func (c *ColorM) Add(other ColorM) {
	// Implementation is just for backward compatibility.
	if c.body == nil {
		c.body = colorMIdentityBody
		c.translate = colorMIdentityTranslate
	}
	if other.body == nil {
		other.body = colorMIdentityBody
		other.translate = colorMIdentityTranslate
	}

	body := make([]float64, len(c.body))
	for i := range c.body {
		body[i] = c.body[i] + other.body[i]
	}

	translate := make([]float64, len(c.translate))
	for i := range c.translate {
		translate[i] = c.translate[i] + other.translate[i]
	}

	c.body = body
	c.translate = translate
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float64) {
	if c.body == nil {
		c.body = []float64{
			r, 0, 0, 0,
			0, g, 0, 0,
			0, 0, b, 0,
			0, 0, 0, a,
		}
		c.translate = colorMIdentityTranslate
		return
	}
	es := make([]float64, len(c.body))
	copy(es, c.body)
	for i := 0; i < ColorMDim-1; i++ {
		es[i] *= r
		es[i+(ColorMDim-1)] *= g
		es[i+(ColorMDim-1)*2] *= b
		es[i+(ColorMDim-1)*3] *= a
	}
	c.body = es

	c.translate = []float64{
		c.translate[0] * r,
		c.translate[1] * g,
		c.translate[2] * b,
		c.translate[3] * a,
	}
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float64) {
	if c.body == nil {
		c.body = colorMIdentityBody
		c.translate = []float64{r, g, b, a}
		return
	}
	es := make([]float64, len(c.translate))
	copy(es, c.translate)
	es[0] += r
	es[1] += g
	es[2] += b
	es[3] += a
	c.translate = es
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = ColorM{
		body: []float64{
			0.2990, 0.5870, 0.1140, 0,
			-0.1687, -0.3313, 0.5000, 0,
			0.5000, -0.4187, -0.0813, 0,
			0, 0, 0, 1,
		},
		translate: []float64{0, 0, 0, 0},
	}
	yCbCrToRgb = ColorM{
		body: []float64{
			1, 0, 1.40200, 0,
			1, -0.34414, -0.71414, 0,
			1, 1.77200, 0, 0,
			0, 0, 0, 1,
		},
		translate: []float64{0, 0, 0, 0},
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
		body: []float64{
			1, 0, 0, 0,
			0, cos, -sin, 0,
			0, sin, cos, 0,
			0, 0, 0, 1,
		},
		translate: []float64{0, 0, 0, 0},
	})
	s := saturationScale
	v := valueScale
	c.Scale(v, s*v, s*v, 1)
	c.Concat(&yCbCrToRgb)
}

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
	colorMIdentityBody = []float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	colorMIdentityTranslate = []float32{
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
// The nil and initial value is identity.
type ColorM struct {
	// When elements is nil, this matrix is identity.
	// elements are immutable and a new array must be created when updating.
	body      []float32
	translate []float32
}

func clamp(x float32) float32 {
	if x > 1 {
		return 1
	}
	if x < 0 {
		return 0
	}
	return x
}

func (c *ColorM) isInited() bool {
	return c != nil && (c.body != nil || c.translate != nil)
}

func (c *ColorM) ScaleOnly() bool {
	if c == nil {
		return true
	}
	if c.body != nil {
		if c.body[1] != 0 {
			return false
		}
		if c.body[2] != 0 {
			return false
		}
		if c.body[3] != 0 {
			return false
		}
		if c.body[4] != 0 {
			return false
		}
		if c.body[6] != 0 {
			return false
		}
		if c.body[7] != 0 {
			return false
		}
		if c.body[8] != 0 {
			return false
		}
		if c.body[9] != 0 {
			return false
		}
		if c.body[11] != 0 {
			return false
		}
		if c.body[12] != 0 {
			return false
		}
		if c.body[13] != 0 {
			return false
		}
		if c.body[14] != 0 {
			return false
		}
	}
	if c.translate != nil {
		for _, e := range c.translate {
			if e != 0 {
				return false
			}
		}
	}
	return true
}

func (c *ColorM) Apply(clr color.Color) color.Color {
	if !c.isInited() {
		return clr
	}
	r, g, b, a := clr.RGBA()
	rf, gf, bf, af := float32(0.0), float32(0.0), float32(0.0), float32(0.0)
	// Unmultiply alpha
	if a > 0 {
		rf = float32(r) / float32(a)
		gf = float32(g) / float32(a)
		bf = float32(b) / float32(a)
		af = float32(a) / 0xffff
	}
	eb := c.body
	if eb == nil {
		eb = colorMIdentityBody
	}
	et := c.translate
	if et == nil {
		et = colorMIdentityTranslate
	}
	rf2 := eb[0]*rf + eb[4]*gf + eb[8]*bf + eb[12]*af + et[0]
	gf2 := eb[1]*rf + eb[5]*gf + eb[9]*bf + eb[13]*af + et[1]
	bf2 := eb[2]*rf + eb[6]*gf + eb[10]*bf + eb[14]*af + et[2]
	af2 := eb[3]*rf + eb[7]*gf + eb[11]*bf + eb[15]*af + et[3]
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

func (c *ColorM) UnsafeElements() ([]float32, []float32) {
	if !c.isInited() {
		return colorMIdentityBody, colorMIdentityTranslate
	}
	eb := c.body
	if eb == nil {
		eb = colorMIdentityBody
	}
	et := c.translate
	if et == nil {
		et = colorMIdentityTranslate
	}
	return eb, et
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float32) *ColorM {
	newC := &ColorM{
		body:      make([]float32, 16),
		translate: make([]float32, 4),
	}
	copy(newC.body, colorMIdentityBody)
	copy(newC.translate, colorMIdentityTranslate)
	if c.isInited() {
		if c.body != nil {
			copy(newC.body, c.body)
		}
		if c.translate != nil {
			copy(newC.translate, c.translate)
		}
	}
	if j < (ColorMDim - 1) {
		newC.body[i+j*(ColorMDim-1)] = element
	} else {
		newC.translate[i] = element
	}
	return newC
}

func (c *ColorM) Equals(other *ColorM) bool {
	if !c.isInited() && !other.isInited() {
		return true
	}

	lhsb := colorMIdentityBody
	lhst := colorMIdentityTranslate
	rhsb := colorMIdentityBody
	rhst := colorMIdentityTranslate
	if other.isInited() {
		if other.body != nil {
			lhsb = other.body
		}
		if other.translate != nil {
			lhst = other.translate
		}
	}
	if c.isInited() {
		if c.body != nil {
			rhsb = c.body
		}
		if c.translate != nil {
			rhst = c.translate
		}
	}
	if &lhsb == &rhsb && &lhst == &rhst {
		return true
	}

	for i := range lhsb {
		if lhsb[i] != rhsb[i] {
			return false
		}
	}
	for i := range lhst {
		if lhst[i] != rhst[i] {
			return false
		}
	}
	return true
}

// Concat multiplies a color matrix with the other color matrix.
// This is same as muptiplying the matrix other and the matrix c in this order.
func (c *ColorM) Concat(other *ColorM) *ColorM {
	if !c.isInited() {
		return other
	}
	if !other.isInited() {
		return c
	}

	lhsb := colorMIdentityBody
	lhst := colorMIdentityTranslate
	rhsb := colorMIdentityBody
	rhst := colorMIdentityTranslate
	if other.isInited() {
		if other.body != nil {
			lhsb = other.body
		}
		if other.translate != nil {
			lhst = other.translate
		}
	}
	if c.isInited() {
		if c.body != nil {
			rhsb = c.body
		}
		if c.translate != nil {
			rhst = c.translate
		}
	}

	return &ColorM{
		// TODO: This is a temporary hack to calculate multiply of transposed matrices.
		// Fix mulSquare implmentation and swap the arguments.
		body: mulSquare(rhsb, lhsb, ColorMDim-1),
		translate: []float32{
			lhsb[0]*rhst[0] + lhsb[4]*rhst[1] + lhsb[8]*rhst[2] + lhsb[12]*rhst[3] + lhst[0],
			lhsb[1]*rhst[0] + lhsb[5]*rhst[1] + lhsb[9]*rhst[2] + lhsb[13]*rhst[3] + lhst[1],
			lhsb[2]*rhst[0] + lhsb[6]*rhst[1] + lhsb[10]*rhst[2] + lhsb[14]*rhst[3] + lhst[2],
			lhsb[3]*rhst[0] + lhsb[7]*rhst[1] + lhsb[11]*rhst[2] + lhsb[15]*rhst[3] + lhst[3],
		},
	}
}

// Add is deprecated.
func (c *ColorM) Add(other *ColorM) *ColorM {
	lhsb := colorMIdentityBody
	lhst := colorMIdentityTranslate
	rhsb := colorMIdentityBody
	rhst := colorMIdentityTranslate
	if other.isInited() {
		if other.body != nil {
			lhsb = other.body
		}
		if other.translate != nil {
			lhst = other.translate
		}
	}
	if c.isInited() {
		if c.body != nil {
			rhsb = c.body
		}
		if c.translate != nil {
			rhst = c.translate
		}
	}

	newC := &ColorM{
		body:      make([]float32, 16),
		translate: make([]float32, 4),
	}
	for i := range lhsb {
		newC.body[i] = lhsb[i] + rhsb[i]
	}
	for i := range lhst {
		newC.translate[i] = lhst[i] + rhst[i]
	}

	return newC
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float32) *ColorM {
	if !c.isInited() {
		return &ColorM{
			body: []float32{
				r, 0, 0, 0,
				0, g, 0, 0,
				0, 0, b, 0,
				0, 0, 0, a,
			},
		}
	}
	eb := make([]float32, len(colorMIdentityBody))
	if c.body != nil {
		copy(eb, c.body)
		for i := 0; i < ColorMDim-1; i++ {
			eb[i*(ColorMDim-1)] *= r
			eb[i*(ColorMDim-1)+1] *= g
			eb[i*(ColorMDim-1)+2] *= b
			eb[i*(ColorMDim-1)+3] *= a
		}
	} else {
		eb[0] = r
		eb[5] = g
		eb[10] = b
		eb[15] = a
	}

	et := make([]float32, len(colorMIdentityTranslate))
	if c.translate != nil {
		et[0] = c.translate[0] * r
		et[1] = c.translate[1] * g
		et[2] = c.translate[2] * b
		et[3] = c.translate[3] * a
	}

	return &ColorM{
		body:      eb,
		translate: et,
	}
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float32) *ColorM {
	if !c.isInited() {
		return &ColorM{
			translate: []float32{r, g, b, a},
		}
	}
	es := make([]float32, len(colorMIdentityTranslate))
	if c.translate != nil {
		copy(es, c.translate)
	}
	es[0] += r
	es[1] += g
	es[2] += b
	es[3] += a
	return &ColorM{
		body:      c.body,
		translate: es,
	}
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = &ColorM{
		body: []float32{
			0.2990, -0.1687, 0.5000, 0,
			0.5870, -0.3313, -0.4187, 0,
			0.1140, 0.5000, -0.0813, 0,
			0, 0, 0, 1,
		},
	}
	yCbCrToRgb = &ColorM{
		body: []float32{
			1, 1, 1, 0,
			0, -0.34414, 1.77200, 0,
			1.40200, -0.71414, 0, 0,
			0, 0, 0, 1,
		},
	}
)

// ChangeHSV changes HSV (Hue-Saturation-Value) elements.
// hueTheta is a radian value to ratate hue.
// saturationScale is a value to scale saturation.
// valueScale is a value to scale value (a.k.a. brightness).
//
// This conversion uses RGB to/from YCrCb conversion.
func (c *ColorM) ChangeHSV(hueTheta float64, saturationScale float32, valueScale float32) *ColorM {
	sin, cos := math.Sincos(hueTheta)
	s32, c32 := float32(sin), float32(cos)
	c = c.Concat(rgbToYCbCr)
	c = c.Concat(&ColorM{
		body: []float32{
			1, 0, 0, 0,
			0, c32, s32, 0,
			0, -s32, c32, 0,
			0, 0, 0, 1,
		},
	})
	s := saturationScale
	v := valueScale
	c = c.Scale(v, s*v, s*v, 1)
	c = c.Concat(yCbCrToRgb)
	return c
}

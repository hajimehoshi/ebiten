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

func (c *ColorM) det() float32 {
	if !c.isInited() {
		return 1
	}

	m00 := c.body[0]
	m01 := c.body[1]
	m02 := c.body[2]
	m03 := c.body[3]
	m10 := c.body[4]
	m11 := c.body[5]
	m12 := c.body[6]
	m13 := c.body[7]
	m20 := c.body[8]
	m21 := c.body[9]
	m22 := c.body[10]
	m23 := c.body[11]
	m30 := c.body[12]
	m31 := c.body[13]
	m32 := c.body[14]
	m33 := c.body[15]

	b234234 := m22*m33 - m23*m32
	b134234 := m21*m33 - m23*m31
	b124234 := m21*m32 - m22*m31
	b034234 := m20*m33 - m23*m30
	b024234 := m20*m32 - m22*m30
	b014234 := m20*m31 - m21*m30

	return m00*(m11*b234234-m12*b134234+m13*b124234) -
		m01*(m10*b234234-m12*b034234+m13*b024234) +
		m02*(m10*b134234-m11*b034234+m13*b014234) -
		m03*(m10*b124234-m11*b024234+m12*b014234)
}

// IsInvertible returns a boolean value indicating
// whether the matrix c is invertible or not.
func (c *ColorM) IsInvertible() bool {
	return c.det() != 0
}

// Invert inverts the matrix.
// If c is not invertible, Invert panics.
func (c *ColorM) Invert() *ColorM {
	if !c.isInited() {
		return nil
	}

	det := c.det()
	if det == 0 {
		panic("affine: c is not invertible")
	}

	m00 := c.body[0]
	m01 := c.body[1]
	m02 := c.body[2]
	m03 := c.body[3]

	m10 := c.body[4]
	m11 := c.body[5]
	m12 := c.body[6]
	m13 := c.body[7]

	m20 := c.body[8]
	m21 := c.body[9]
	m22 := c.body[10]
	m23 := c.body[11]

	m30 := c.body[12]
	m31 := c.body[13]
	m32 := c.body[14]
	m33 := c.body[15]

	m40 := c.translate[0]
	m41 := c.translate[1]
	m42 := c.translate[2]
	m43 := c.translate[3]

	a2334 := m32*m43 - m33*m42
	a1334 := m31*m43 - m33*m41
	a1234 := m31*m42 - m32*m41
	a0334 := m30*m43 - m33*m40
	a0234 := m30*m42 - m32*m40
	a0134 := m30*m41 - m31*m40
	a2324 := m22*m43 - m23*m42
	a1324 := m21*m43 - m23*m41
	a1224 := m21*m42 - m22*m41
	a0324 := m20*m43 - m23*m40
	a0224 := m20*m42 - m22*m40
	a0124 := m20*m41 - m21*m40

	b234234 := m22*m33 - m23*m32
	b134234 := m21*m33 - m23*m31
	b124234 := m21*m32 - m22*m31
	b123234 := m21*a2334 - m22*a1334 + m23*a1234
	b034234 := m20*m33 - m23*m30
	b024234 := m20*m32 - m22*m30
	b023234 := m20*a2334 - m22*a0334 + m23*a0234
	b014234 := m20*m31 - m21*m30
	b013234 := m20*a1334 - m21*a0334 + m23*a0134
	b012234 := m20*a1234 - m21*a0234 + m22*a0134
	b234134 := m12*m33 - m13*m32
	b134134 := m11*m33 - m13*m31
	b124134 := m11*m32 - m12*m31
	b123134 := m11*a2334 - m12*a1334 + m13*a1234
	b234124 := m12*m23 - m13*m22
	b134124 := m11*m23 - m13*m21
	b124124 := m11*m22 - m12*m21
	b123124 := m11*a2324 - m12*a1324 + m13*a1224
	b034134 := m10*m33 - m13*m30
	b024134 := m10*m32 - m12*m30
	b023134 := m10*a2334 - m12*a0334 + m13*a0234
	b034124 := m10*m23 - m13*m20
	b024124 := m10*m22 - m12*m20
	b023124 := m10*a2324 - m12*a0324 + m13*a0224
	b014134 := m10*m31 - m11*m30
	b013134 := m10*a1334 - m11*a0334 + m13*a0134
	b014124 := m10*m21 - m11*m20
	b013124 := m10*a1324 - m11*a0324 + m13*a0124
	b012134 := m10*a1234 - m11*a0234 + m12*a0134
	b012124 := m10*a1224 - m11*a0224 + m12*a0124

	m := &ColorM{
		body:      make([]float32, 16),
		translate: make([]float32, 4),
	}

	idet := 1 / det

	m.body[0] = idet * (m11*b234234 - m12*b134234 + m13*b124234)
	m.body[1] = idet * -(m01*b234234 - m02*b134234 + m03*b124234)
	m.body[2] = idet * (m01*b234134 - m02*b134134 + m03*b124134)
	m.body[3] = idet * -(m01*b234124 - m02*b134124 + m03*b124124)
	m.body[4] = idet * -(m10*b234234 - m12*b034234 + m13*b024234)
	m.body[5] = idet * (m00*b234234 - m02*b034234 + m03*b024234)
	m.body[6] = idet * -(m00*b234134 - m02*b034134 + m03*b024134)
	m.body[7] = idet * (m00*b234124 - m02*b034124 + m03*b024124)
	m.body[8] = idet * (m10*b134234 - m11*b034234 + m13*b014234)
	m.body[9] = idet * -(m00*b134234 - m01*b034234 + m03*b014234)
	m.body[10] = idet * (m00*b134134 - m01*b034134 + m03*b014134)
	m.body[11] = idet * -(m00*b134124 - m01*b034124 + m03*b014124)
	m.body[12] = idet * -(m10*b124234 - m11*b024234 + m12*b014234)
	m.body[13] = idet * (m00*b124234 - m01*b024234 + m02*b014234)
	m.body[14] = idet * -(m00*b124134 - m01*b024134 + m02*b014134)
	m.body[15] = idet * (m00*b124124 - m01*b024124 + m02*b014124)
	m.translate[0] = idet * (m10*b123234 - m11*b023234 + m12*b013234 - m13*b012234)
	m.translate[1] = idet * -(m00*b123234 - m01*b023234 + m02*b013234 - m03*b012234)
	m.translate[2] = idet * (m00*b123134 - m01*b023134 + m02*b013134 - m03*b012134)
	m.translate[3] = idet * -(m00*b123124 - m01*b023124 + m02*b013124 - m03*b012124)
	return m
}

// Element returns a value of a matrix at (i, j).
func (c *ColorM) Element(i, j int) float32 {
	b, t := c.UnsafeElements()
	if j < ColorMDim-1 {
		return b[i+j*(ColorMDim-1)]
	}
	return t[i]
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

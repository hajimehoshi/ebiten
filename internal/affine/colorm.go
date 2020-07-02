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
	if c == nil {
		return 0
	}
	if !c.isInited() {
		return 1
	}

	m00 := c.body[0]
	m01 := c.body[1]
	m02 := c.body[2]
	m03 := c.body[3]
	m04 := float32(0)
	m10 := c.body[4]
	m11 := c.body[5]
	m12 := c.body[6]
	m13 := c.body[7]
	m14 := float32(0)
	m20 := c.body[8]
	m21 := c.body[9]
	m22 := c.body[10]
	m23 := c.body[11]
	m24 := float32(0)
	m30 := c.body[12]
	m31 := c.body[13]
	m32 := c.body[14]
	m33 := c.body[15]
	m34 := float32(0)
	m40 := c.translate[0]
	m41 := c.translate[1]
	m42 := c.translate[2]
	m43 := c.translate[3]
	m44 := float32(1)

	a3434 := m33*m44 - m34*m43
	a2434 := m32*m44 - m34*m42
	a2334 := m32*m43 - m33*m42
	a1434 := m31*m44 - m34*m41
	a1334 := m31*m43 - m33*m41
	a1234 := m31*m42 - m32*m41
	a0434 := m30*m44 - m34*m40
	a0334 := m30*m43 - m33*m40
	a0234 := m30*m42 - m32*m40
	a0134 := m30*m41 - m31*m40

	b234234 := m22*a3434 - m23*a2434 + m24*a2334
	b134234 := m21*a3434 - m23*a1434 + m24*a1334
	b124234 := m21*a2434 - m22*a1434 + m24*a1234
	b123234 := m21*a2334 - m22*a1334 + m23*a1234
	b034234 := m20*a3434 - m23*a0434 + m24*a0334
	b024234 := m20*a2434 - m22*a0434 + m24*a0234
	b023234 := m20*a2334 - m22*a0334 + m23*a0234
	b014234 := m20*a1434 - m21*a0434 + m24*a0134
	b013234 := m20*a1334 - m21*a0334 + m23*a0134
	b012234 := m20*a1234 - m21*a0234 + m22*a0134

	det := m00*(m11*b234234-m12*b134234+
		m13*b124234-m14*b123234) -
		m01*(m10*b234234-m12*b034234+
			m13*b024234-m14*b023234) +
		m02*(m10*b134234-m11*b034234+
			m13*b014234-m14*b013234) -
		m03*(m10*b124234-m11*b024234+
			m12*b014234-m14*b012234) +
		m04*(m10*b123234-m11*b023234+
			m12*b013234-m13*b012234)
	if det == 0 {
		return 0
	}
	return 1 / det
}

// IsInvertible returns a boolean value indicating
// whether the matrix g is invertible or not.
func (c *ColorM) IsInvertible() bool {
	return c.det() != 0
}

// Invert inverts the matrix.
// If c is not invertible, Invert panics.
func (c *ColorM) Invert() {
	det := c.det()
	if det == 0 {
		panic("ebiten: c is not invertible")
	}

	if !c.isInited() {
		c.body = make([]float32, 16)
		c.translate = make([]float32, 4)
		copy(c.body, colorMIdentityBody)
		copy(c.translate, colorMIdentityTranslate)
	}

	m00 := c.body[0]
	m01 := c.body[1]
	m02 := c.body[2]
	m03 := c.body[3]
	m04 := float32(0)
	m10 := c.body[4]
	m11 := c.body[5]
	m12 := c.body[6]
	m13 := c.body[7]
	m14 := float32(0)
	m20 := c.body[8]
	m21 := c.body[9]
	m22 := c.body[10]
	m23 := c.body[11]
	m24 := float32(0)
	m30 := c.body[12]
	m31 := c.body[13]
	m32 := c.body[14]
	m33 := c.body[15]
	m34 := float32(0)
	m40 := c.translate[0]
	m41 := c.translate[1]
	m42 := c.translate[2]
	m43 := c.translate[3]
	m44 := float32(1)

	a3434 := m33*m44 - m34*m43
	a2434 := m32*m44 - m34*m42
	a2334 := m32*m43 - m33*m42
	a1434 := m31*m44 - m34*m41
	a1334 := m31*m43 - m33*m41
	a1234 := m31*m42 - m32*m41
	a0434 := m30*m44 - m34*m40
	a0334 := m30*m43 - m33*m40
	a0234 := m30*m42 - m32*m40
	a0134 := m30*m41 - m31*m40
	a3424 := m23*m44 - m24*m43
	a2424 := m22*m44 - m24*m42
	a2324 := m22*m43 - m23*m42
	a1424 := m21*m44 - m24*m41
	a1324 := m21*m43 - m23*m41
	a1224 := m21*m42 - m22*m41
	// a3423 := m23*m34 - m24*m33 // unused (due to virtual column [j=4])
	// a2423 := m22*m34 - m24*m32 // unused (due to virtual column [j=4])
	// a2323 := m22*m33 - m23*m32 // unused (due to virtual column [j=4])
	// a1423 := m21*m34 - m24*m31 // unused (due to virtual column [j=4])
	// a1323 := m21*m33 - m23*m31 // unused (due to virtual column [j=4])
	// a1223 := m21*m32 - m22*m31 // unused (due to virtual column [j=4])
	a0424 := m20*m44 - m24*m40
	a0324 := m20*m43 - m23*m40
	a0224 := m20*m42 - m22*m40
	// a0423 := m20*m34 - m24*m30 // unused (due to virtual column [j=4])
	// a0323 := m20*m33 - m23*m30 // unused (due to virtual column [j=4])
	// a0223 := m20*m32 - m22*m30 // unused (due to virtual column [j=4])
	a0124 := m20*m41 - m21*m40
	// a0123 := m20*m31 - m21*m30 // unused (due to virtual column [j=4])

	b234234 := m22*a3434 - m23*a2434 + m24*a2334
	b134234 := m21*a3434 - m23*a1434 + m24*a1334
	b124234 := m21*a2434 - m22*a1434 + m24*a1234
	b123234 := m21*a2334 - m22*a1334 + m23*a1234
	b034234 := m20*a3434 - m23*a0434 + m24*a0334
	b024234 := m20*a2434 - m22*a0434 + m24*a0234
	b023234 := m20*a2334 - m22*a0334 + m23*a0234
	b014234 := m20*a1434 - m21*a0434 + m24*a0134
	b013234 := m20*a1334 - m21*a0334 + m23*a0134
	b012234 := m20*a1234 - m21*a0234 + m22*a0134
	b234134 := m12*a3434 - m13*a2434 + m14*a2334
	b134134 := m11*a3434 - m13*a1434 + m14*a1334
	b124134 := m11*a2434 - m12*a1434 + m14*a1234
	b123134 := m11*a2334 - m12*a1334 + m13*a1234
	b234124 := m12*a3424 - m13*a2424 + m14*a2324
	b134124 := m11*a3424 - m13*a1424 + m14*a1324
	b124124 := m11*a2424 - m12*a1424 + m14*a1224
	b123124 := m11*a2324 - m12*a1324 + m13*a1224
	//b234123 := m12*a3423 - m13*a2423 + m14*a2323 // unused (virtual column)
	//b134123 := m11*a3423 - m13*a1423 + m14*a1323 // unused (virtual column)
	//b124123 := m11*a2423 - m12*a1423 + m14*a1223 // unused (virtual column)
	//b123123 := m11*a2323 - m12*a1323 + m13*a1223 // unused (virtual column)
	b034134 := m10*a3434 - m13*a0434 + m14*a0334
	b024134 := m10*a2434 - m12*a0434 + m14*a0234
	b023134 := m10*a2334 - m12*a0334 + m13*a0234
	b034124 := m10*a3424 - m13*a0424 + m14*a0324
	b024124 := m10*a2424 - m12*a0424 + m14*a0224
	b023124 := m10*a2324 - m12*a0324 + m13*a0224
	// b034123 := m10*a3423 - m13*a0423 + m14*a0323 // unused (virtual column)
	// b024123 := m10*a2423 - m12*a0423 + m14*a0223 // unused (virtual column)
	// b023123 := m10*a2323 - m12*a0323 + m13*a0223 // unused (virtual column)
	b014134 := m10*a1434 - m11*a0434 + m14*a0134
	b013134 := m10*a1334 - m11*a0334 + m13*a0134
	b014124 := m10*a1424 - m11*a0424 + m14*a0124
	b013124 := m10*a1324 - m11*a0324 + m13*a0124
	// b014123 := m10*a1423 - m11*a0423 + m14*a0123 // unused (virtual column)
	// b013123 := m10*a1323 - m11*a0323 + m13*a0123 // unused (virtual column)
	b012134 := m10*a1234 - m11*a0234 + m12*a0134
	b012124 := m10*a1224 - m11*a0224 + m12*a0124
	// b012123 := m10*a1223 - m11*a0223 + m12*a0123 // unused (virtual column)

	c.body[0] = det * (m11*b234234 - m12*b134234 + m13*b124234 - m14*b123234)
	c.body[1] = det * -(m01*b234234 - m02*b134234 + m03*b124234 - m04*b123234)
	c.body[2] = det * (m01*b234134 - m02*b134134 + m03*b124134 - m04*b123134)
	c.body[3] = det * -(m01*b234124 - m02*b134124 + m03*b124124 - m04*b123124)
	// the fifth column is discarded because it's not used by colorm:
	// xm04 = det *   ( m01 * b234123 - m02 * b134123 + m03 * b124123 - m04 * b123123 )
	c.body[4] = det * -(m10*b234234 - m12*b034234 + m13*b024234 - m14*b023234)
	c.body[5] = det * (m00*b234234 - m02*b034234 + m03*b024234 - m04*b023234)
	c.body[6] = det * -(m00*b234134 - m02*b034134 + m03*b024134 - m04*b023134)
	c.body[7] = det * (m00*b234124 - m02*b034124 + m03*b024124 - m04*b023124)
	// the fifth column is discarded because it's not used by colorm:
	// xm14 = det * - ( m00 * b234123 - m02 * b034123 + m03 * b024123 - m04 * b023123 )
	c.body[8] = det * (m10*b134234 - m11*b034234 + m13*b014234 - m14*b013234)
	c.body[9] = det * -(m00*b134234 - m01*b034234 + m03*b014234 - m04*b013234)
	c.body[10] = det * (m00*b134134 - m01*b034134 + m03*b014134 - m04*b013134)
	c.body[11] = det * -(m00*b134124 - m01*b034124 + m03*b014124 - m04*b013124)
	// the fifth column is discarded because it's not used by colorm:
	// xm24 = det *   ( m00 * b134123 - m01 * b034123 + m03 * b014123 - m04 * b013123 )
	c.body[12] = det * -(m10*b124234 - m11*b024234 + m12*b014234 - m14*b012234)
	c.body[13] = det * (m00*b124234 - m01*b024234 + m02*b014234 - m04*b012234)
	c.body[14] = det * -(m00*b124134 - m01*b024134 + m02*b014134 - m04*b012134)
	c.body[15] = det * (m00*b124124 - m01*b024124 + m02*b014124 - m04*b012124)
	// the fifth column is discarded because it's not used by colorm:
	// xm34 = det * - ( m00 * b124123 - m01 * b024123 + m02 * b014123 - m04 * b012123 )
	c.translate[0] = det * (m10*b123234 - m11*b023234 + m12*b013234 - m13*b012234)
	c.translate[1] = det * -(m00*b123234 - m01*b023234 + m02*b013234 - m03*b012234)
	c.translate[2] = det * (m00*b123134 - m01*b023134 + m02*b013134 - m03*b012134)
	c.translate[3] = det * -(m00*b123124 - m01*b023124 + m02*b013124 - m03*b012124)
	// the fifth column is discarded because it's not used by colorm:
	// xm44 = det *   ( m00 * b123123 - m01 * b023123 + m02 * b013123 - m03 * b012123 )
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

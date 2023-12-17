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
	"fmt"
	"image/color"
	"math"
)

// ColorMDim is a dimension of a ColorM.
const ColorMDim = 5

var (
	colorMIdentityBody = [...]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	colorMIdentityTranslate = [...]float32{
		0, 0, 0, 0,
	}
)

// ColorM represents a matrix to transform coloring when rendering an image.
//
// ColorM is applied to the source alpha color
// while an Image's pixels' format is alpha premultiplied.
// Before applying a matrix, a color is un-multiplied, and after applying the matrix,
// the color is multiplied again.
type ColorM interface {
	String() string

	IsIdentity() bool
	ScaleOnly() bool
	At(i, j int) float32
	Elements(body []float32, translate []float32)
	Apply(clr color.Color) color.Color

	// IsInvertible returns a boolean value indicating
	// whether the matrix c is invertible or not.
	IsInvertible() bool

	// Invert inverts the matrix.
	// If c is not invertible, Invert panics.
	Invert() ColorM

	Equals(other ColorM) bool

	// Concat multiplies a color matrix with the other color matrix.
	// This is same as multiplying the matrix other and the matrix c in this order.
	Concat(other ColorM) ColorM

	// Scale scales the matrix by (r, g, b, a).
	Scale(r, g, b, a float32) ColorM

	// Translate translates the matrix by (r, g, b, a).
	Translate(r, g, b, a float32) ColorM

	scaleElements() (r, g, b, a float32)
}

func colorMString(c ColorM) string {
	var b [16]float32
	var t [4]float32
	c.Elements(b[:], t[:])
	return fmt.Sprintf("[[%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f], [%f, %f, %f, %f, %f]]",
		b[0], b[4], b[8], b[12], t[0],
		b[1], b[5], b[9], b[13], t[1],
		b[2], b[6], b[10], b[14], t[2],
		b[3], b[7], b[11], b[15], t[3])
}

type ColorMIdentity struct{}

type colorMImplScale struct {
	scale [4]float32
}

type colorMImplBodyTranslate struct {
	body      [16]float32
	translate [4]float32
}

func (c ColorMIdentity) String() string {
	return "Identity[]"
}

func (c colorMImplScale) String() string {
	return fmt.Sprintf("Scale[%f, %f, %f, %f]", c.scale[0], c.scale[1], c.scale[2], c.scale[3])
}

func (c *colorMImplBodyTranslate) String() string {
	return colorMString(c)
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

func (c ColorMIdentity) IsIdentity() bool {
	return true
}

func (c colorMImplScale) IsIdentity() bool {
	return c.scale == [4]float32{1, 1, 1, 1}
}

func (c *colorMImplBodyTranslate) IsIdentity() bool {
	return c.body == colorMIdentityBody && c.translate == colorMIdentityTranslate
}

func (c ColorMIdentity) ScaleOnly() bool {
	return true
}

func (c colorMImplScale) ScaleOnly() bool {
	return true
}

func (c *colorMImplBodyTranslate) ScaleOnly() bool {
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
	for _, e := range c.translate {
		if e != 0 {
			return false
		}
	}
	return true
}

func (c ColorMIdentity) scaleElements() (r, g, b, a float32) {
	return 1, 1, 1, 1
}

func (c colorMImplScale) scaleElements() (r, g, b, a float32) {
	return c.scale[0], c.scale[1], c.scale[2], c.scale[3]
}

func (c *colorMImplBodyTranslate) scaleElements() (r, g, b, a float32) {
	return c.body[0], c.body[5], c.body[10], c.body[15]
}

func colorToFloat32s(clr color.Color) (float32, float32, float32, float32) {
	r, g, b, a := clr.RGBA()
	rf, gf, bf, af := float32(0.0), float32(0.0), float32(0.0), float32(0.0)
	// Unmultiply alpha
	if a > 0 {
		rf = float32(r) / float32(a)
		gf = float32(g) / float32(a)
		bf = float32(b) / float32(a)
		af = float32(a) / 0xffff
	}
	return rf, gf, bf, af
}

func (c ColorMIdentity) Apply(clr color.Color) color.Color {
	rf, gf, bf, af := colorToFloat32s(clr)
	return color.NRGBA64{
		R: uint16(rf * 0xffff),
		G: uint16(gf * 0xffff),
		B: uint16(bf * 0xffff),
		A: uint16(af * 0xffff),
	}
}

func (c colorMImplScale) Apply(clr color.Color) color.Color {
	rf, gf, bf, af := colorToFloat32s(clr)
	rf *= c.scale[0]
	gf *= c.scale[1]
	bf *= c.scale[2]
	af *= c.scale[3]
	rf = clamp(rf)
	gf = clamp(gf)
	bf = clamp(bf)
	af = clamp(af)
	return color.NRGBA64{
		R: uint16(rf * 0xffff),
		G: uint16(gf * 0xffff),
		B: uint16(bf * 0xffff),
		A: uint16(af * 0xffff),
	}
}

func (c *colorMImplBodyTranslate) Apply(clr color.Color) color.Color {
	rf, gf, bf, af := colorToFloat32s(clr)
	eb := &c.body
	et := &c.translate
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

func (c ColorMIdentity) Elements(body []float32, translate []float32) {
	body[0] = 1
	body[1] = 0
	body[2] = 0
	body[3] = 0
	body[4] = 0
	body[5] = 1
	body[6] = 0
	body[7] = 0
	body[8] = 0
	body[9] = 0
	body[10] = 1
	body[11] = 0
	body[12] = 0
	body[13] = 0
	body[14] = 0
	body[15] = 1
	translate[0] = 0
	translate[1] = 0
	translate[2] = 0
	translate[3] = 0
}

func (c colorMImplScale) Elements(body []float32, translate []float32) {
	body[0] = c.scale[0]
	body[1] = 0
	body[2] = 0
	body[3] = 0
	body[4] = 0
	body[5] = c.scale[1]
	body[6] = 0
	body[7] = 0
	body[8] = 0
	body[9] = 0
	body[10] = c.scale[2]
	body[11] = 0
	body[12] = 0
	body[13] = 0
	body[14] = 0
	body[15] = c.scale[3]
	translate[0] = 0
	translate[1] = 0
	translate[2] = 0
	translate[3] = 0
}

func (c *colorMImplBodyTranslate) Elements(body []float32, translate []float32) {
	copy(body, c.body[:])
	copy(translate, c.translate[:])
}

func (c *colorMImplBodyTranslate) det() float32 {
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

func (c ColorMIdentity) IsInvertible() bool {
	return true
}

func (c colorMImplScale) IsInvertible() bool {
	return c.scale[0] != 0 && c.scale[1] != 0 && c.scale[2] != 0 && c.scale[3] != 0
}

func (c *colorMImplBodyTranslate) IsInvertible() bool {
	return c.det() != 0
}

func (c ColorMIdentity) Invert() ColorM {
	return c
}

func (c colorMImplScale) Invert() ColorM {
	return colorMImplScale{
		scale: [4]float32{
			1 / c.scale[0],
			1 / c.scale[1],
			1 / c.scale[2],
			1 / c.scale[3],
		},
	}
}

func (c *colorMImplBodyTranslate) Invert() ColorM {
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

	m := &colorMImplBodyTranslate{
		body: colorMIdentityBody,
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

func (c ColorMIdentity) At(i, j int) float32 {
	if i == j {
		return 1
	}
	return 0
}

func (c colorMImplScale) At(i, j int) float32 {
	if i == j {
		return c.scale[i]
	}
	return 0
}

func (c *colorMImplBodyTranslate) At(i, j int) float32 {
	if j < ColorMDim-1 {
		return c.body[i+j*(ColorMDim-1)]
	}
	return c.translate[i]
}

// ColorMSetElement sets an element at (i, j).
func ColorMSetElement(c ColorM, i, j int, element float32) ColorM {
	newImpl := &colorMImplBodyTranslate{
		body: colorMIdentityBody,
	}
	if !c.IsIdentity() {
		c.Elements(newImpl.body[:], newImpl.translate[:])
	}
	if j < (ColorMDim - 1) {
		newImpl.body[i+j*(ColorMDim-1)] = element
	} else {
		newImpl.translate[i] = element
	}
	return newImpl
}

func (c ColorMIdentity) Equals(other ColorM) bool {
	return other.IsIdentity()
}

func (c colorMImplScale) Equals(other ColorM) bool {
	if !other.ScaleOnly() {
		return false
	}

	r, g, b, a := other.scaleElements()
	if c.scale[0] != r {
		return false
	}
	if c.scale[1] != g {
		return false
	}
	if c.scale[2] != b {
		return false
	}
	if c.scale[3] != a {
		return false
	}
	return true
}

func (c *colorMImplBodyTranslate) Equals(other ColorM) bool {
	var lhsb [16]float32
	var lhst [4]float32

	// Calling a method of an interface type escapes arguments to heap (#2119).
	// Instead, cast `other` to a concrete type and call `Elements` functions of it.
	switch other := other.(type) {
	case ColorMIdentity:
		other.Elements(lhsb[:], lhst[:])
	case colorMImplScale:
		other.Elements(lhsb[:], lhst[:])
	case *colorMImplBodyTranslate:
		other.Elements(lhsb[:], lhst[:])
	default:
		panic("affine: unexpected ColorM implementation")
	}

	return lhsb == c.body && lhst == c.translate
}

func (c ColorMIdentity) Concat(other ColorM) ColorM {
	return other
}

func (c colorMImplScale) Concat(other ColorM) ColorM {
	if other.IsIdentity() {
		return c
	}

	if other.ScaleOnly() {
		return c.Scale(other.scaleElements())
	}

	var lhsb [16]float32
	var lhst [4]float32

	// Calling a method of an interface type escapes arguments to heap (#2119).
	// Instead, cast `other` to a concrete type and call `Elements` functions of it.
	switch other := other.(type) {
	case ColorMIdentity:
		other.Elements(lhsb[:], lhst[:])
	case colorMImplScale:
		other.Elements(lhsb[:], lhst[:])
	case *colorMImplBodyTranslate:
		other.Elements(lhsb[:], lhst[:])
	default:
		panic("affine: unexpected ColorM implementation")
	}

	s := &c.scale
	return &colorMImplBodyTranslate{
		body: [...]float32{
			lhsb[0] * s[0], lhsb[1] * s[0], lhsb[2] * s[0], lhsb[3] * s[0],
			lhsb[4] * s[1], lhsb[5] * s[1], lhsb[6] * s[1], lhsb[7] * s[1],
			lhsb[8] * s[2], lhsb[9] * s[2], lhsb[10] * s[2], lhsb[11] * s[2],
			lhsb[12] * s[3], lhsb[13] * s[3], lhsb[14] * s[3], lhsb[15] * s[3],
		},
		translate: lhst,
	}
}

func (c *colorMImplBodyTranslate) Concat(other ColorM) ColorM {
	if other.IsIdentity() {
		return c
	}

	var lhsb [16]float32
	var lhst [4]float32

	// Calling a method of an interface type escapes arguments to heap (#2119).
	// Instead, cast `other` to a concrete type and call `Elements` functions of it.
	switch other := other.(type) {
	case ColorMIdentity:
		other.Elements(lhsb[:], lhst[:])
	case colorMImplScale:
		other.Elements(lhsb[:], lhst[:])
	case *colorMImplBodyTranslate:
		other.Elements(lhsb[:], lhst[:])
	default:
		panic("affine: unexpected ColorM implementation")
	}

	rhsb := &c.body
	rhst := &c.translate

	return &colorMImplBodyTranslate{
		// TODO: This is a temporary hack to calculate multiply of transposed matrices.
		// Fix mulSquare implementation and swap the arguments.
		body: mulSquare(rhsb, &lhsb, ColorMDim-1),
		translate: [...]float32{
			lhsb[0]*rhst[0] + lhsb[4]*rhst[1] + lhsb[8]*rhst[2] + lhsb[12]*rhst[3] + lhst[0],
			lhsb[1]*rhst[0] + lhsb[5]*rhst[1] + lhsb[9]*rhst[2] + lhsb[13]*rhst[3] + lhst[1],
			lhsb[2]*rhst[0] + lhsb[6]*rhst[1] + lhsb[10]*rhst[2] + lhsb[14]*rhst[3] + lhst[2],
			lhsb[3]*rhst[0] + lhsb[7]*rhst[1] + lhsb[11]*rhst[2] + lhsb[15]*rhst[3] + lhst[3],
		},
	}
}

func (c ColorMIdentity) Scale(r, g, b, a float32) ColorM {
	if r == 1 && g == 1 && b == 1 && a == 1 {
		return c
	}

	return colorMImplScale{
		scale: [...]float32{r, g, b, a},
	}
}

func (c colorMImplScale) Scale(r, g, b, a float32) ColorM {
	if r == 1 && g == 1 && b == 1 && a == 1 {
		return c
	}

	return colorMImplScale{
		scale: [...]float32{
			c.scale[0] * r,
			c.scale[1] * g,
			c.scale[2] * b,
			c.scale[3] * a,
		},
	}
}

func (c *colorMImplBodyTranslate) Scale(r, g, b, a float32) ColorM {
	if r == 1 && g == 1 && b == 1 && a == 1 {
		return c
	}

	if c.ScaleOnly() {
		sr, sg, sb, sa := c.scaleElements()
		return colorMImplScale{
			scale: [...]float32{r * sr, g * sg, b * sb, a * sa},
		}
	}

	eb := c.body
	for i := 0; i < ColorMDim-1; i++ {
		eb[i*(ColorMDim-1)] *= r
		eb[i*(ColorMDim-1)+1] *= g
		eb[i*(ColorMDim-1)+2] *= b
		eb[i*(ColorMDim-1)+3] *= a
	}

	et := [...]float32{
		c.translate[0] * r,
		c.translate[1] * g,
		c.translate[2] * b,
		c.translate[3] * a,
	}

	return &colorMImplBodyTranslate{
		body:      eb,
		translate: et,
	}
}

func (c ColorMIdentity) Translate(r, g, b, a float32) ColorM {
	if r == 0 && g == 0 && b == 0 && a == 0 {
		return c
	}

	return &colorMImplBodyTranslate{
		body:      colorMIdentityBody,
		translate: [...]float32{r, g, b, a},
	}
}

func (c colorMImplScale) Translate(r, g, b, a float32) ColorM {
	if r == 0 && g == 0 && b == 0 && a == 0 {
		return c
	}

	return &colorMImplBodyTranslate{
		body: [...]float32{
			c.scale[0], 0, 0, 0,
			0, c.scale[1], 0, 0,
			0, 0, c.scale[2], 0,
			0, 0, 0, c.scale[3],
		},
		translate: [...]float32{r, g, b, a},
	}
}

func (c *colorMImplBodyTranslate) Translate(r, g, b, a float32) ColorM {
	if r == 0 && g == 0 && b == 0 && a == 0 {
		return c
	}

	es := c.translate
	es[0] += r
	es[1] += g
	es[2] += b
	es[3] += a
	return &colorMImplBodyTranslate{
		body:      c.body,
		translate: es,
	}
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = &colorMImplBodyTranslate{
		body: [...]float32{
			0.2990, -0.1687, 0.5000, 0,
			0.5870, -0.3313, -0.4187, 0,
			0.1140, 0.5000, -0.0813, 0,
			0, 0, 0, 1,
		},
	}
	yCbCrToRgb = &colorMImplBodyTranslate{
		body: [...]float32{
			1, 1, 1, 0,
			0, -0.34414, 1.77200, 0,
			1.40200, -0.71414, 0, 0,
			0, 0, 0, 1,
		},
	}
)

// ChangeHSV changes HSV (Hue-Saturation-Value) elements.
// hueTheta is a radian value to rotate hue.
// saturationScale is a value to scale saturation.
// valueScale is a value to scale value (a.k.a. brightness).
//
// This conversion uses RGB to/from YCrCb conversion.
func ChangeHSV(c ColorM, hueTheta float64, saturationScale float32, valueScale float32) ColorM {
	if hueTheta == 0 && saturationScale == 1 {
		v := valueScale
		return c.Scale(v, v, v, 1)
	}

	sin, cos := math.Sincos(hueTheta)
	s32, c32 := float32(sin), float32(cos)
	c = c.Concat(rgbToYCbCr)
	c = c.Concat(&colorMImplBodyTranslate{
		body: [...]float32{
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

type cachedScalingColorMKey struct {
	r, g, b, a float32
}

type cachedScalingColorMValue struct {
	c     *colorMImplScale
	atime uint64
}

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
	colorMIdentity = []float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
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
	// Note that elements are transposed for OpenGL.
	elements []float32
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
	return c != nil && c.elements != nil
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
	es := c.elements
	if es == nil {
		es = colorMIdentity
	}
	rf2 := es[0]*rf + es[4]*gf + es[8]*bf + es[12]*af + es[16]
	gf2 := es[1]*rf + es[5]*gf + es[9]*bf + es[13]*af + es[17]
	bf2 := es[2]*rf + es[6]*gf + es[10]*bf + es[14]*af + es[18]
	af2 := es[3]*rf + es[7]*gf + es[11]*bf + es[15]*af + es[19]
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

func (c *ColorM) UnsafeElements() []float32 {
	if !c.isInited() {
		return colorMIdentity
	}
	return c.elements
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float32) *ColorM {
	newC := &ColorM{
		elements: make([]float32, 20),
	}
	copy(newC.elements, colorMIdentity)
	if c.isInited() {
		copy(newC.elements, c.elements)
	}
	newC.elements[i+j*(ColorMDim-1)] = element
	return newC
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

	lhs := colorMIdentity
	rhs := colorMIdentity
	if other.isInited() {
		lhs = other.elements
	}
	if c.isInited() {
		rhs = c.elements
	}

	return &ColorM{
		elements: mulAffine(lhs, rhs, ColorMDim),
	}
}

// Add is deprecated.
func (c *ColorM) Add(other *ColorM) *ColorM {
	lhs := colorMIdentity
	rhs := colorMIdentity
	if other.isInited() {
		lhs = other.elements
	}
	if c.isInited() {
		rhs = c.elements
	}

	newC := &ColorM{
		elements: make([]float32, 20),
	}
	for i := range lhs {
		newC.elements[i] = lhs[i] + rhs[i]
	}

	return newC
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float32) *ColorM {
	if !c.isInited() {
		return &ColorM{
			elements: []float32{
				r, 0, 0, 0,
				0, g, 0, 0,
				0, 0, b, 0,
				0, 0, 0, a,
				0, 0, 0, 0,
			},
		}
	}
	es := make([]float32, len(colorMIdentity))
	copy(es, c.elements)
	for i := 0; i < ColorMDim; i++ {
		es[i*(ColorMDim-1)] *= r
		es[i*(ColorMDim-1)+1] *= g
		es[i*(ColorMDim-1)+2] *= b
		es[i*(ColorMDim-1)+3] *= a
	}

	return &ColorM{
		elements: es,
	}
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float32) *ColorM {
	if !c.isInited() {
		return &ColorM{
			elements: []float32{
				1, 0, 0, 0,
				0, 1, 0, 0,
				0, 0, 1, 0,
				0, 0, 0, 1,
				r, g, b, a,
			},
		}
	}
	es := make([]float32, len(colorMIdentity))
	copy(es, c.elements)
	es[16] += r
	es[17] += g
	es[18] += b
	es[19] += a
	return &ColorM{
		elements: es,
	}
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = &ColorM{
		elements: []float32{
			0.2990, -0.1687, 0.5000, 0,
			0.5870, -0.3313, -0.4187, 0,
			0.1140, 0.5000, -0.0813, 0,
			0, 0, 0, 1,
			0, 0, 0, 0,
		},
	}
	yCbCrToRgb = &ColorM{
		elements: []float32{
			1, 1, 1, 0,
			0, -0.34414, 1.77200, 0,
			1.40200, -0.71414, 0, 0,
			0, 0, 0, 1,
			0, 0, 0, 0,
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
		elements: []float32{
			1, 0, 0, 0,
			0, c32, s32, 0,
			0, -s32, c32, 0,
			0, 0, 0, 1,
			0, 0, 0, 0,
		},
	})
	s := saturationScale
	v := valueScale
	c = c.Scale(v, s*v, s*v, 1)
	c = c.Concat(yCbCrToRgb)
	return c
}

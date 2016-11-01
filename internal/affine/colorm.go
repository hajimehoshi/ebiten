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
	"math"
)

// ColorMDim is a dimension of a ColorM.
const ColorMDim = 5

// A ColorM represents a matrix to transform coloring when rendering an image.
//
// A ColorM is applied to the source alpha color
// while an Image's pixels' format is alpha premultiplied.
// Before applying a matrix, a color is un-multiplied, and after applying the matrix,
// the color is multiplied again.
//
// The initial value is identity.
type ColorM struct {
	initialized bool
	es          [(ColorMDim - 1) * (ColorMDim)]float64
}

func (c *ColorM) dim() int {
	return ColorMDim
}

func (c *ColorM) initialize() {
	c.initialized = true
	for i := 0; i < ColorMDim-1; i++ {
		c.es[i*ColorMDim+i] = 1
	}
}

// Element returns a value of a matrix at (i, j).
func (c *ColorM) Element(i, j int) float64 {
	if !c.initialized {
		if i == j {
			return 1
		}
		return 0
	}
	return c.es[i*ColorMDim+j]
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, element float64) {
	if !c.initialized {
		c.initialize()
	}
	c.es[i*ColorMDim+j] = element
}

func (c *ColorM) Equals(other *ColorM) bool {
	if !c.initialized {
		c.initialize()
	}
	if !other.initialized {
		other.initialize()
	}
	return c.es == other.es
}

// Concat multiplies a color matrix with the other color matrix.
// This is same as muptiplying the matrix other and the matrix c in this order.
func (c *ColorM) Concat(other ColorM) {
	result := ColorM{}
	mul(&other, c, &result)
	*c = result
}

// Add adds a color matrix with the other color matrix.
func (c *ColorM) Add(other ColorM) {
	result := ColorM{}
	add(&other, c, &result)
	*c = result
}

// Scale scales the matrix by (r, g, b, a).
func (c *ColorM) Scale(r, g, b, a float64) {
	for i := 0; i < ColorMDim-1; i++ {
		c.SetElement(0, i, c.Element(0, i)*r)
		c.SetElement(1, i, c.Element(1, i)*g)
		c.SetElement(2, i, c.Element(2, i)*b)
		c.SetElement(3, i, c.Element(3, i)*a)
	}
}

// Translate translates the matrix by (r, g, b, a).
func (c *ColorM) Translate(r, g, b, a float64) {
	c.SetElement(0, 4, c.Element(0, 4)+r)
	c.SetElement(1, 4, c.Element(1, 4)+g)
	c.SetElement(2, 4, c.Element(2, 4)+b)
	c.SetElement(3, 4, c.Element(3, 4)+a)
}

// RotateHue rotates the hue.
func (c *ColorM) RotateHue(theta float64) {
	c.ChangeHSV(theta, 1, 1)
}

var (
	// The YCbCr value ranges are:
	//   Y:  [ 0   - 1  ]
	//   Cb: [-0.5 - 0.5]
	//   Cr: [-0.5 - 0.5]

	rgbToYCbCr = ColorM{
		initialized: true,
		es: [...]float64{
			0.2990, 0.5870, 0.1140, 0, 0,
			-0.1687, -0.3313, 0.5000, 0, 0,
			0.5000, -0.4187, -0.0813, 0, 0,
			0, 0, 0, 1, 0,
		},
	}
	yCbCrToRgb = ColorM{
		initialized: true,
		es: [...]float64{
			1, 0, 1.40200, 0, 0,
			1, -0.34414, -0.71414, 0, 0,
			1, 1.77200, 0, 0, 0,
			0, 0, 0, 1, 0,
		},
	}
)

// ChangeHSV changes HSV (Hue-Saturation-Value) values.
// hueTheta is a radian value to ratate hue.
// saturationScale is a value to scale saturation.
// valueScale is a value to scale value (a.k.a. brightness).
//
// This conversion uses RGB to/from YCrCb conversion.
func (c *ColorM) ChangeHSV(hueTheta float64, saturationScale float64, valueScale float64) {
	sin, cos := math.Sincos(hueTheta)
	c.Concat(rgbToYCbCr)
	c.Concat(ColorM{
		initialized: true,
		es: [...]float64{
			1, 0, 0, 0, 0,
			0, cos, -sin, 0, 0,
			0, sin, cos, 0, 0,
			0, 0, 0, 1, 0,
		},
	})
	s := saturationScale
	v := valueScale
	c.Scale(v, s*v, s*v, 1)
	c.Concat(yCbCrToRgb)
}

var monochrome ColorM

func init() {
	monochrome.ChangeHSV(0, 0, 1)
}

// Monochrome returns a color matrix to make an image monochrome.
func Monochrome() ColorM {
	return monochrome
}

// ScaleColor is deprecated as of 1.2.0-alpha. Use Scale instead.
func ScaleColor(r, g, b, a float64) ColorM {
	return ColorM{
		initialized: true,
		es: [...]float64{
			r, 0, 0, 0, 0,
			0, g, 0, 0, 0,
			0, 0, b, 0, 0,
			0, 0, 0, a, 0,
		},
	}
}

// TranslateColor is deprecated as of 1.2.0-alpha. Use Translate instead.
func TranslateColor(r, g, b, a float64) ColorM {
	return ColorM{
		initialized: true,
		es: [...]float64{
			1, 0, 0, 0, r,
			0, 1, 0, 0, g,
			0, 0, 1, 0, b,
			0, 0, 0, 1, a,
		},
	}
}

// RotateHue is deprecated as of 1.2.0-alpha. Use RotateHue member function instead.
func RotateHue(theta float64) ColorM {
	c := ColorM{}
	c.RotateHue(theta)
	return c
}

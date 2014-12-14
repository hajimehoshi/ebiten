/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"image/color"
	"math"
)

// ColorMatrixDim is a dimension of a ColorMatrix.
const ColorMatrixDim = 5

// A ColorMatrix represents a matrix to transform coloring when rendering a texture or a render target.
type ColorMatrix struct {
	Elements [ColorMatrixDim - 1][ColorMatrixDim]float64
}

// ColorMatrixI returns an identity color matrix.
func ColorMatrixI() ColorMatrix {
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{1, 0, 0, 0, 0},
			{0, 1, 0, 0, 0},
			{0, 0, 1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func (c *ColorMatrix) dim() int {
	return ColorMatrixDim
}

// Element returns a value of a matrix at (i, j).
func (c *ColorMatrix) Element(i, j int) float64 {
	return c.Elements[i][j]
}

// Concat multiplies a color matrix with the other color matrix.
func (c *ColorMatrix) Concat(other ColorMatrix) {
	result := ColorMatrix{}
	mul(&other, c, &result)
	*c = result
}

// IsIdentity returns a boolean indicating whether the color matrix is an identity.
func (c *ColorMatrix) IsIdentity() bool {
	return isIdentity(c)
}

func (c *ColorMatrix) setElement(i, j int, element float64) {
	c.Elements[i][j] = element
}

// Monochrome returns a color matrix to make an image monochrome.
func Monochrome() ColorMatrix {
	const r = 6968.0 / 32768.0
	const g = 23434.0 / 32768.0
	const b = 2366.0 / 32768.0
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

// ScaleColor returns a color matrix that scales a color matrix by clr.
func ScaleColor(clr color.Color) ColorMatrix {
	rf, gf, bf, af := rgba(clr)
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{rf, 0, 0, 0, 0},
			{0, gf, 0, 0, 0},
			{0, 0, bf, 0, 0},
			{0, 0, 0, af, 0},
		},
	}
}

// TranslateColor returns a color matrix that translates a color matrix by clr.
func (c *ColorMatrix) Translate(clr color.Color) ColorMatrix {
	rf, gf, bf, af := rgba(clr)
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{1, 0, 0, 0, rf},
			{0, 1, 0, 0, gf},
			{0, 0, 1, 0, bf},
			{0, 0, 0, 1, af},
		},
	}
}

// RotateHue returns a color matrix to rotate the hue
func RotateHue(theta float64) ColorMatrix {
	sin, cos := math.Sincos(theta)
	v1 := cos + (1.0-cos)/3.0
	v2 := 1.0/3.0*(1.0-cos) - math.Sqrt(1.0/3.0)*sin
	v3 := 1.0/3.0*(1.0-cos) + math.Sqrt(1.0/3.0)*sin
	// TODO: Need to clamp the values between 0 and 1?
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{v1, v2, v3, 0, 0},
			{v3, v1, v2, 0, 0},
			{v2, v3, v1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func rgba(clr color.Color) (float64, float64, float64, float64) {
	r, g, b, a := clr.RGBA()
	rf := float64(r) / float64(math.MaxUint16)
	gf := float64(g) / float64(math.MaxUint16)
	bf := float64(b) / float64(math.MaxUint16)
	af := float64(a) / float64(math.MaxUint16)
	return rf, gf, bf, af
}

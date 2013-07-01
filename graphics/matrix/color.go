// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package matrix

const colorDim = 5

type Color struct {
	Elements [colorDim - 1][colorDim]float64
}

func IdentityColor() Color {
	return Color{
		[colorDim - 1][colorDim]float64{
			{1, 0, 0, 0, 0},
			{0, 1, 0, 0, 0},
			{0, 0, 1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func (matrix *Color) Dim() int {
	return colorDim
}

func (matrix *Color) Concat(other Color) {
	result := Color{}
	mul(&other, matrix, &result)
	*matrix = result
}

func (matrix *Color) IsIdentity() bool {
	return isIdentity(matrix)
}

func (matrix *Color) element(i, j int) float64 {
	return matrix.Elements[i][j]
}

func (matrix *Color) setElement(i, j int, element float64) {
	matrix.Elements[i][j] = element
}

func Monochrome() Color {
	const r float64 = 6968.0 / 32768.0
	const g float64 = 23434.0 / 32768.0
	const b float64 = 2366.0 / 32768.0
	return Color{
		[colorDim - 1][colorDim]float64{
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

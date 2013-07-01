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

type Affine interface {
	Dim() int
	element(i, j int) float64
	setElement(i, j int, element float64)
}

func isIdentity(matrix Affine) bool {
	dim := matrix.Dim()
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := matrix.element(i, j)
			if i == j && element != 1 {
				return false
			} else if i != j && element != 0 {
				return false
			}
		}
	}
	return true
}

func mul(lhs, rhs, result Affine) {
	dim := lhs.Dim()
	if dim != rhs.Dim() {
		panic("diffrent-sized matrices can't be multiplied")
	}

	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := float64(0)
			for k := 0; k < dim-1; k++ {
				element += lhs.element(i, k) *
					rhs.element(k, j)
			}
			if j == dim-1 {
				element += lhs.element(i, j)
			}
			result.setElement(i, j, element)
		}
	}
}

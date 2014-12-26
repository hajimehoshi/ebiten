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

package ebiten

type affine interface {
	dim() int
	Element(i, j int) float64
	SetElement(i, j int, element float64)
}

func isIdentity(ebiten affine) bool {
	dim := ebiten.dim()
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := ebiten.Element(i, j)
			if i == j && element != 1 {
				return false
			} else if i != j && element != 0 {
				return false
			}
		}
	}
	return true
}

func add(lhs, rhs, result affine) {
	dim := lhs.dim()
	if dim != rhs.dim() {
		panic("diffrent-sized matrices can't be multiplied")
	}

	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			v := lhs.Element(i, j) + rhs.Element(i, j)
			result.SetElement(i, j, v)
		}
	}
}

func mul(lhs, rhs, result affine) {
	dim := lhs.dim()
	if dim != rhs.dim() {
		panic("diffrent-sized matrices can't be multiplied")
	}

	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := float64(0)
			for k := 0; k < dim-1; k++ {
				element += lhs.Element(i, k) *
					rhs.Element(k, j)
			}
			if j == dim-1 {
				element += lhs.Element(i, j)
			}
			result.SetElement(i, j, element)
		}
	}
}

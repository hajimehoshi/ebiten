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

func affineAt(es []float32, i, j int, dim int) float32 {
	if i == dim-1 {
		if j == dim-1 {
			return 1
		}
		return 0
	}
	return es[i+j*(dim-1)]
}

func mulAffine(lhs, rhs []float32, dim int) []float32 {
	result := make([]float32, len(lhs))
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			e := float32(0.0)
			for k := 0; k < dim; k++ {
				e += affineAt(lhs, i, k, dim) * affineAt(rhs, k, j, dim)
			}
			result[i+j*(dim-1)] = e
		}
	}
	return result
}

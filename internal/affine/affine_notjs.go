// Copyright 2017 The Ebiten Authors
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

// +build !js

package affine

import (
	"math"

	"github.com/hajimehoshi/ebiten/internal/endian"
)

func elements(values string, dim int) []float64 {
	result := make([]float64, dim*(dim-1))
	if values == "" {
		for i := 0; i < dim-1; i++ {
			result[i*dim+i] = 1
		}
		return result
	}
	if endian.IsLittle() {
		for i := 0; i < len(values)/8; i++ {
			v := uint64(0)
			for k := 7; 0 <= k; k-- {
				v <<= 8
				v += uint64(values[i*8+k])
			}
			result[i] = math.Float64frombits(v)
		}
	} else {
		for i := 0; i < len(values)/8; i++ {
			v := uint64(0)
			for k := 0; k < 8; k++ {
				v <<= 8
				v += uint64(values[i*8+k])
			}
			result[i] = math.Float64frombits(v)
		}
	}
	return result
}

func setElements(values []float64, dim int) string {
	result := make([]uint8, len(values)*8)
	for i, v := range values {
		copy(result[i*8:(i+1)*8], uint64ToBytes(math.Float64bits(v)))
	}
	return string(result)
}

func setElement(values string, dim int, i, j int, value float64) string {
	if values == "" {
		values = identityValues[dim]
	}
	b := uint64ToBytes(math.Float64bits(value))
	offset := 8 * (i*dim + j)
	return values[:offset] + string(b) + values[offset+8:]
}

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

func element(values string, dim int, i, j int) float64 {
	if values == "" {
		if i == j {
			return 1
		}
		return 0
	}
	offset := 8 * (i*dim + j)
	v := uint64(0)
	if endian.IsLittle() {
		for k := 7; 0 <= k; k-- {
			v <<= 8
			v += uint64(values[offset+k])
		}
	} else {
		for k := 0; k < 8; k++ {
			v <<= 8
			v += uint64(values[offset+k])
		}
	}
	return math.Float64frombits(v)
}

func setElement(values string, dim int, i, j int, value float64) string {
	if values == "" {
		values = identityValues[dim]
	}
	b := uint64ToBytes(math.Float64bits(value))
	offset := 8 * (i*dim + j)
	return values[:offset] + string(b) + values[offset+8:]
}

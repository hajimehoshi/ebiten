// Copyright 2018 The Ebiten Authors
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

package graphics

import (
	"math"
)

// MipmapLevel returns an appropriate mipmap level for the given determinant of a geometry matrix.
//
// MipmapLevel returns -1 if det is 0.
//
// MipmapLevel panics if det is NaN.
func MipmapLevel(det float32) int {
	if math.IsNaN(float64(det)) {
		panic("graphicsutil: det must be finite")
	}
	if det == 0 {
		return -1
	}

	d := math.Abs(float64(det))
	level := 0
	for d < 0.25 {
		level++
		d *= 4
	}
	return level
}
